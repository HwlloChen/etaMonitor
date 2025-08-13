package monitor

import (
	"log"
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
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:                   db,
		config:               cfg,
		playerSessionService: services.NewPlayerSessionService(db),
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

	// 会话清理任务 (每5分钟执行一次)
	sessionCleanupTicker := time.NewTicker(5 * time.Minute)
	defer sessionCleanupTicker.Stop()

	log.Printf("Server monitoring started with interval: %v", s.config.MonitorInterval)

	for {
		select {
		case <-ticker.C:
			s.checkAllServers()
		case <-cleanupTicker.C:
			s.cleanupOldStats()
		case <-sessionCleanupTicker.C:
			s.playerSessionService.CleanupInactiveSessions()
		}
	}
}

func (s *Service) checkAllServers() {
	var servers []models.Server
	if err := s.db.Find(&servers).Error; err != nil {
		log.Printf("Failed to fetch servers: %v", err)
		return
	}

	for _, server := range servers {
		go s.checkServer(&server)
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
			"status":         "online",
			"players_online": serverInfo.Players.Online,
			"max_players":    serverInfo.Players.Max,
			"ping":           serverInfo.Ping,
			"version":        serverInfo.Version.Name,
			"motd":           extractDescriptionText(serverInfo.Description),
			"last_checked":   &stat.Timestamp,
		}
		// 保存这些信息作为最后一次在线状态
		s.db.Model(server).Update("last_online_data", serverInfo)
		s.db.Model(server).Updates(serverUpdates)

		// 广播服务器状态更新
		s.broadcastServerStatus(server.ID, map[string]interface{}{
			"id":             server.ID,
			"name":           server.Name,
			"status":         "online",
			"players_online": serverInfo.Players.Online,
			"max_players":    serverInfo.Players.Max,
			"ping":           serverInfo.Ping,
			"version":        serverInfo.Version.Name,
			"motd":           extractDescriptionText(serverInfo.Description),
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
			"status":         "offline",
			"last_checked":   &stat.Timestamp,
			"players_online": 0,
			"max_players":    0,
			"ping":           -1,
		})

		// 广播服务器离线状态
		s.broadcastServerStatus(server.ID, map[string]interface{}{
			"id":     server.ID,
			"name":   server.Name,
			"status": "offline",
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
