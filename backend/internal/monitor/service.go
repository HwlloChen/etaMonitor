package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"etamonitor/internal/config"
	"etamonitor/internal/models"
	"etamonitor/internal/services"
	"etamonitor/internal/websocket"

	"gorm.io/gorm"
)

type Service struct {
	db                   *gorm.DB
	config               *config.Config
	playerSessionService *services.PlayerSessionService
	
	// 并发控制
	semaphore            chan struct{} // 控制并发goroutine数量
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 限制最大并发检查数为10个
	maxConcurrent := 10
	
	return &Service{
		db:                   db,
		config:               cfg,
		playerSessionService: services.NewPlayerSessionService(db),
		semaphore:            make(chan struct{}, maxConcurrent),
		ctx:                  ctx,
		cancel:               cancel,
	}
}

func (s *Service) Start() {
	ticker := time.NewTicker(s.config.MonitorInterval)
	defer ticker.Stop()

	// 启动时清理旧数据
	s.cleanupOldStats()

	// 定期清理任务 (每小时执行一次)
	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	log.Printf("Server monitoring started with interval: %v", s.config.MonitorInterval)

	for {
		select {
		case <-s.ctx.Done():
			// 等待所有goroutine完成
			log.Println("Waiting for all monitoring goroutines to finish...")
			s.wg.Wait()
			log.Println("Monitor service stopped")
			return
		case <-ticker.C:
			s.checkAllServers()
		case <-cleanupTicker.C:
			s.cleanupOldStats()
		}
	}
}

// Stop 优雅停止监控服务
func (s *Service) Stop() {
	s.cancel()
}

func (s *Service) checkAllServers() {
	var servers []models.Server
	if err := s.db.Find(&servers).Error; err != nil {
		log.Printf("Failed to fetch servers: %v", err)
		return
	}

	log.Printf("Checking %d servers...", len(servers))
	
	for _, server := range servers {
		// 复制server变量避免闭包问题
		serverCopy := server
		
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in server check for %s: %v", serverCopy.Name, r)
				}
			}()
			
			s.checkServerWithTimeout(&serverCopy)
		}()
	}
}

func (s *Service) checkServerWithTimeout(server *models.Server) {
	// 获取信号量，限制并发数
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-s.ctx.Done():
		return
	}
	
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.checkServer(server)
	}()
	
	select {
	case <-done:
		// 正常完成
	case <-ctx.Done():
		log.Printf("Server check timeout for %s", server.Name)
	}
}

func (s *Service) checkServer(server *models.Server) {
	var serverInfo *services.MinecraftServer
	var err error

	// 根据服务器类型进行ping
	switch server.Type {
	case "java":
		serverInfo, err = services.JavaServerPing(server.Address, server.Port)
	case "bedrock":
		serverInfo, err = services.BedrockServerPing(server.Address, server.Port)
	case "auto":
		var detectedType string
		serverInfo, detectedType, err = services.AutoDetectServer(
			server.Address,
			server.Port,
			19132, // 基岩版默认端口
		)
		if err == nil && detectedType != "" {
			// 更新检测到的服务器类型
			s.db.Model(server).Update("type", detectedType)
		}
	default:
		// 默认尝试Java版
		serverInfo, err = services.JavaServerPing(server.Address, server.Port)
	}

	// 创建统计记录
	stat := models.ServerStat{
		ServerID:  server.ID,
		Timestamp: time.Now(),
	}

	// 确定服务器状态
	wasOnline := server.Status == "online"

	if err == nil && serverInfo != nil {
		stat.PlayersOnline = serverInfo.Players.Online
		stat.MaxPlayers = serverInfo.Players.Max
		stat.Ping = serverInfo.Ping
		stat.Version = serverInfo.Version.Name
		stat.MOTD = extractDescriptionText(serverInfo.Description)

		// 更新玩家会话记录
		s.playerSessionService.UpdatePlayerSessions(server, serverInfo.Players.Sample)

		// 更新服务器的实时信息
		serverUpdates := map[string]interface{}{
			"status":          "online",
			"players_online":  serverInfo.Players.Online,
			"max_players":     serverInfo.Players.Max,
			"anonymous_count": s.playerSessionService.GetAnonymousCount(server.ID),
			"ping":            serverInfo.Ping,
			"version":         serverInfo.Version.Name,
			"motd":            extractDescriptionText(serverInfo.Description),
			"last_checked":    &stat.Timestamp,
		}
		// 保存这些信息作为最后一次在线状态
		s.db.Model(server).Update("last_online_data", serverInfo)
		s.db.Model(server).Updates(serverUpdates)

		// 广播服务器状态更新
		s.broadcastServerStatus(server.ID, map[string]interface{}{
			"id":              server.ID,
			"name":            server.Name,
			"status":          "online",
			"players_online":  serverInfo.Players.Online,
			"max_players":     serverInfo.Players.Max,
			"anonymous_count": s.playerSessionService.GetAnonymousCount(server.ID),
			"ping":            serverInfo.Ping,
			"version":         serverInfo.Version.Name,
			"motd":            extractDescriptionText(serverInfo.Description),
		})
	} else {
		log.Printf("Failed to ping server %s: %v", server.Name, err)
		stat.Ping = -1

		// 服务器离线时，使用玩家会话服务清理会话
		if wasOnline {
			s.playerSessionService.UpdatePlayerSessions(server, []services.PlayerInfo{})
		}

		// 更新离线状态和相关数据
		s.db.Model(server).Updates(map[string]interface{}{
			"status":          "offline",
			"last_checked":    &stat.Timestamp,
			"players_online":  0,
			"max_players":     0,
			"anonymous_count": 0,
			"ping":            -1,
		})

		// 广播服务器离线状态
		s.broadcastServerStatus(server.ID, map[string]interface{}{
			"id":              server.ID,
			"name":            server.Name,
			"status":          "offline",
			"anonymous_count": 0,
		})
	}

	// 保存统计数据
	if err := s.db.Create(&stat).Error; err != nil {
		log.Printf("Failed to save server stat for %s: %v", server.Name, err)
	}
}

// broadcastServerStatus 广播服务器状态更新
func (s *Service) broadcastServerStatus(serverID uint, data map[string]interface{}) {
	websocket.BroadcastServerStatus(serverID, data)
}

// extractDescriptionText 从Description结构体中提取文本
func extractDescriptionText(desc services.Description) string {
	if desc.Text != "" {
		return desc.Text
	}
	if len(desc.Extra) > 0 {
		result := ""
		for _, extra := range desc.Extra {
			result += extra.Text
		}
		return result
	}
	return "Minecraft Server"
}

// cleanupOldStats 清理超过30天的旧统计数据
func (s *Service) cleanupOldStats() {
	cutoff := time.Now().AddDate(0, 0, -30) // 30天前

	result := s.db.Where("timestamp < ?", cutoff).Delete(&models.ServerStat{})
	if result.Error != nil {
		log.Printf("Failed to cleanup old stats: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("Cleaned up %d old stat records", result.RowsAffected)
	}
}