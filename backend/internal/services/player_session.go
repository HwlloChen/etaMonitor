package services

import (
	"log"
	"sync"
	"time"

	"etamonitor/internal/models"
	"etamonitor/internal/websocket"
	"gorm.io/gorm"
)

// PlayerSessionService 玩家会话服务
type PlayerSessionService struct {
	db            *gorm.DB
	lastPlayerMap map[uint]map[string]bool // serverID -> playerName -> exists
	mutex         sync.RWMutex             // 保护lastPlayerMap的并发访问
}

// NewPlayerSessionService 创建玩家会话服务
func NewPlayerSessionService(db *gorm.DB) *PlayerSessionService {
	service := &PlayerSessionService{
		db:            db,
		lastPlayerMap: make(map[uint]map[string]bool),
	}
	
	// 服务启动时清理未结束的会话并初始化状态
	service.initializeService()
	
	return service
}

// initializeService 初始化服务状态
func (p *PlayerSessionService) initializeService() {
	log.Println("初始化玩家会话服务...")
	
	// 1. 清理所有未结束的会话（服务重启意味着所有玩家都已离线）
	cutoff := time.Now().Add(-10 * time.Minute) // 给10分钟的缓冲时间
	
	var staleSessions []models.PlayerSession
	p.db.Where("leave_time IS NULL AND join_time < ?", cutoff).Find(&staleSessions)
	
	for _, session := range staleSessions {
		now := time.Now()
		duration := int(now.Sub(session.JoinTime).Seconds())
		
		session.LeaveTime = &now
		session.Duration = duration
		p.db.Save(&session)
		
		// 更新玩家总在线时间
		var player models.Player
		if p.db.First(&player, session.PlayerID).Error == nil {
			player.TotalPlaytime += duration
			player.LastSeen = now
			p.db.Save(&player)
		}
		
		log.Printf("清理过期会话: 玩家ID=%d, 服务器ID=%d, 持续时间=%d秒", 
			session.PlayerID, session.ServerID, duration)
	}
	
	// 2. 初始化所有服务器的玩家映射为空（因为服务重启，所有玩家都离线了）
	var servers []models.Server
	p.db.Find(&servers)
	
	p.mutex.Lock()
	for _, server := range servers {
		p.lastPlayerMap[server.ID] = make(map[string]bool)
		log.Printf("初始化服务器 %s 的玩家状态", server.Name)
	}
	p.mutex.Unlock()
	
	log.Println("玩家会话服务初始化完成")
}

// UpdatePlayerSessions 更新玩家会话状态
func (p *PlayerSessionService) UpdatePlayerSessions(server *models.Server, currentPlayers []PlayerInfo) {
	serverID := server.ID
	
	// 获取当前服务器的玩家状态（带锁保护）
	p.mutex.Lock()
	if p.lastPlayerMap[serverID] == nil {
		p.lastPlayerMap[serverID] = make(map[string]bool)
	}
	lastPlayerMap := make(map[string]bool)
	for k, v := range p.lastPlayerMap[serverID] {
		lastPlayerMap[k] = v
	}
	p.mutex.Unlock()
	
	currentPlayerMap := make(map[string]bool)
	
	// 处理当前在线玩家
	var newPlayers []PlayerInfo
	for _, playerInfo := range currentPlayers {
		playerName := playerInfo.Name
		currentPlayerMap[playerName] = true
		
		// 检查是否是新加入的玩家
		if !lastPlayerMap[playerName] {
			newPlayers = append(newPlayers, playerInfo)
		}
	}
	
	// 检查离开的玩家
	var leftPlayers []string
	for playerName := range lastPlayerMap {
		if !currentPlayerMap[playerName] {
			leftPlayers = append(leftPlayers, playerName)
		}
	}
	
	// 处理玩家加入事件（不持有锁）
	for _, player := range newPlayers {
		p.handlePlayerJoin(server, player.Name, player.ID)
	}
	
	// 处理玩家离开事件（不持有锁）
	for _, playerName := range leftPlayers {
		p.handlePlayerLeave(server, playerName)
	}
	
	// 更新玩家映射（带锁保护）
	p.mutex.Lock()
	p.lastPlayerMap[serverID] = currentPlayerMap
	p.mutex.Unlock()
}

// handlePlayerJoin 处理玩家加入
func (p *PlayerSessionService) handlePlayerJoin(server *models.Server, playerName, playerUUID string) {
	log.Printf("玩家加入: %s -> %s", playerName, server.Name)
	
	// 查找或创建玩家记录
	player := p.findOrCreatePlayer(playerName, playerUUID)
	if player == nil {
		log.Printf("创建玩家记录失败: %s", playerName)
		return
	}
	
	// 检查是否已有活跃会话（防止重复创建）
	var existingSession models.PlayerSession
	err := p.db.Where("player_id = ? AND server_id = ? AND leave_time IS NULL", 
		player.ID, server.ID).First(&existingSession).Error
	
	if err == nil {
		log.Printf("玩家 %s 在服务器 %s 已有活跃会话，跳过创建", playerName, server.Name)
		return
	}
	
	// 创建新的会话记录
	session := &models.PlayerSession{
		PlayerID: player.ID,
		ServerID: server.ID,
		JoinTime: time.Now(),
	}
	
	if err := p.db.Create(session).Error; err != nil {
		log.Printf("创建玩家会话失败: %v", err)
		return
	}
	
	// 更新玩家的最后在线时间
	player.LastSeen = time.Now()
	p.db.Save(player)
	
	// 保存玩家活动记录
	p.savePlayerActivity(player.ID, server.ID, "join", 0)
	
	// 发送实时通知
	p.broadcastPlayerJoin(server.ID, player, server.Name)
}

// handlePlayerLeave 处理玩家离开
func (p *PlayerSessionService) handlePlayerLeave(server *models.Server, playerName string) {
	log.Printf("玩家离开: %s -> %s", playerName, server.Name)
	
	// 查找玩家
	var player models.Player
	if err := p.db.Where("username = ?", playerName).First(&player).Error; err != nil {
		log.Printf("查找玩家失败: %v", err)
		return
	}
	
	// 查找当前活跃的会话
	var session models.PlayerSession
	if err := p.db.Where("player_id = ? AND server_id = ? AND leave_time IS NULL", 
		player.ID, server.ID).First(&session).Error; err != nil {
		log.Printf("查找活跃会话失败: %v", err)
		return
	}
	
	// 结束会话
	now := time.Now()
	duration := int(now.Sub(session.JoinTime).Seconds())
	
	session.LeaveTime = &now
	session.Duration = duration
	
	if err := p.db.Save(&session).Error; err != nil {
		log.Printf("更新会话失败: %v", err)
		return
	}
	
	// 更新玩家的总在线时间
	player.TotalPlaytime += duration
	player.LastSeen = now
	p.db.Save(&player)
	
	// 更新玩家等级
	p.updatePlayerRank(&player)
	
	// 保存玩家活动记录
	p.savePlayerActivity(player.ID, server.ID, "leave", duration)
	
	// 发送实时通知
	p.broadcastPlayerLeave(server.ID, &player, server.Name, duration)
}

// findOrCreatePlayer 查找或创建玩家记录
func (p *PlayerSessionService) findOrCreatePlayer(username, uuid string) *models.Player {
	var player models.Player
	
	// 首先尝试通过UUID查找
	if uuid != "" {
		err := p.db.Where("uuid = ?", uuid).First(&player).Error
		if err == nil {
			// 找到了，更新用户名（可能会变化）
			if player.Username != username {
				player.Username = username
				p.db.Save(&player)
			}
			return &player
		}
	}
	
	// 通过用户名查找
	err := p.db.Where("username = ?", username).First(&player).Error
	if err == nil {
		// 找到了，更新UUID（如果之前没有）
		if uuid != "" && player.UUID == "" {
			player.UUID = uuid
			p.db.Save(&player)
		}
		return &player
	}
	
	// 创建新玩家
	now := time.Now()
	player = models.Player{
		Username:      username,
		UUID:          uuid,
		FirstSeen:     now,
		LastSeen:      now,
		TotalPlaytime: 0,
		Rank:          "Newcomer",
	}
	
	if err := p.db.Create(&player).Error; err != nil {
		log.Printf("创建玩家失败: %v", err)
		return nil
	}
	
	return &player
}

// updatePlayerRank 更新玩家等级
func (p *PlayerSessionService) updatePlayerRank(player *models.Player) {
	playtimeHours := float64(player.TotalPlaytime) / 3600.0
	
	var newRank string
	switch {
	case playtimeHours >= 500:
		newRank = "Legend"
	case playtimeHours >= 200:
		newRank = "Master"
	case playtimeHours >= 100:
		newRank = "Expert"
	case playtimeHours >= 50:
		newRank = "Veteran"
	case playtimeHours >= 20:
		newRank = "Regular"
	case playtimeHours >= 5:
		newRank = "Member"
	default:
		newRank = "Newcomer"
	}
	
	if player.Rank != newRank {
		oldRank := player.Rank
		player.Rank = newRank
		p.db.Save(player)
		
		log.Printf("玩家 %s 等级提升: %s -> %s (游戏时间: %.1f小时)", 
			player.Username, oldRank, newRank, playtimeHours)
		
		// TODO: 可以在这里添加称号系统的检查
		p.checkAndAwardTitles(player)
	}
}

// checkAndAwardTitles 检查并授予称号
func (p *PlayerSessionService) checkAndAwardTitles(player *models.Player) {
	// 获取玩家的在线时间统计
	var sessions []models.PlayerSession
	p.db.Where("player_id = ? AND leave_time IS NOT NULL", player.ID).Find(&sessions)
	
	// 分析在线时间模式
	nightOwlSessions := 0  // 夜猫子 (22:00-06:00)
	earlyBirdSessions := 0 // 早鸟 (06:00-10:00)
	weekendSessions := 0   // 周末战士
	
	for _, session := range sessions {
		hour := session.JoinTime.Hour()
		weekday := session.JoinTime.Weekday()
		
		// 检查夜猫子模式
		if hour >= 22 || hour < 6 {
			nightOwlSessions++
		}
		
		// 检查早鸟模式
		if hour >= 6 && hour < 10 {
			earlyBirdSessions++
		}
		
		// 检查周末模式
		if weekday == time.Saturday || weekday == time.Sunday {
			weekendSessions++
		}
	}
	
	totalSessions := len(sessions)
	if totalSessions > 0 {
		// 夜猫子称号 (30%以上的会话在夜晚)
		if float64(nightOwlSessions)/float64(totalSessions) >= 0.3 {
			p.awardTitle(player.ID, "夜猫子")
		}
		
		// 早鸟称号 (20%以上的会话在早晨)
		if float64(earlyBirdSessions)/float64(totalSessions) >= 0.2 {
			p.awardTitle(player.ID, "早鸟")
		}
		
		// 周末战士称号 (40%以上的会话在周末)
		if float64(weekendSessions)/float64(totalSessions) >= 0.4 {
			p.awardTitle(player.ID, "周末战士")
		}
	}
	
	// 基于总在线时间的称号
	playtimeHours := float64(player.TotalPlaytime) / 3600.0
	if playtimeHours >= 100 {
		p.awardTitle(player.ID, "时间管理大师")
	}
	if playtimeHours >= 1000 {
		p.awardTitle(player.ID, "传奇玩家")
	}
}

// awardTitle 授予称号
func (p *PlayerSessionService) awardTitle(playerID uint, title string) {
	// 检查是否已经有这个称号
	var existingTitle models.PlayerTitle
	err := p.db.Where("player_id = ? AND title = ?", playerID, title).First(&existingTitle).Error
	if err == nil {
		return // 已经有这个称号
	}
	
	// 创建新称号
	newTitle := models.PlayerTitle{
		PlayerID: playerID,
		Title:    title,
		EarnedAt: time.Now(),
	}
	
	if err := p.db.Create(&newTitle).Error; err != nil {
		log.Printf("授予称号失败: %v", err)
		return
	}
	
	log.Printf("玩家获得新称号: 玩家ID=%d, 称号=%s", playerID, title)
}

// savePlayerActivity 保存玩家活动记录
func (p *PlayerSessionService) savePlayerActivity(playerID uint, serverID uint, activityType string, sessionDuration int) {
	activity := models.PlayerActivity{
		PlayerID:        playerID,
		ServerID:        serverID,
		ActivityType:    activityType,
		Timestamp:       time.Now(),
		SessionDuration: sessionDuration,
	}
	
	if err := p.db.Create(&activity).Error; err != nil {
		log.Printf("保存玩家活动记录失败: %v", err)
	}
}

// broadcastPlayerJoin 广播玩家加入消息
func (p *PlayerSessionService) broadcastPlayerJoin(serverID uint, player *models.Player, serverName string) {
	data := map[string]interface{}{
		"username":       player.Username,
		"uuid":           player.UUID,
		"server_name":    serverName,
		"rank":           player.Rank,
		"avatar":         p.getPlayerAvatar(player.Username),
		"players_online": p.getCurrentPlayersCount(serverID),
	}
	
	websocket.BroadcastPlayerJoin(serverID, data)
}

// broadcastPlayerLeave 广播玩家离开消息
func (p *PlayerSessionService) broadcastPlayerLeave(serverID uint, player *models.Player, serverName string, duration int) {
	data := map[string]interface{}{
		"username":        player.Username,
		"uuid":            player.UUID,
		"server_name":     serverName,
		"rank":            player.Rank,
		"session_duration": duration,
		"avatar":          p.getPlayerAvatar(player.Username),
		"players_online":  p.getCurrentPlayersCount(serverID),
	}
	
	websocket.BroadcastPlayerLeave(serverID, data)
}

// getPlayerAvatar 获取玩家头像URL
func (p *PlayerSessionService) getPlayerAvatar(name string) string {
	if name == "" {
		return "https://crafthead.net/avatar/steve"
	}
	return "https://crafthead.net/avatar/" + name
}

// getCurrentPlayersCount 获取当前在线玩家数
func (p *PlayerSessionService) getCurrentPlayersCount(serverID uint) int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	if playerMap, exists := p.lastPlayerMap[serverID]; exists {
		return len(playerMap)
	}
	return 0
}

// GetActivePlayerSessions 获取活跃的玩家会话
func (p *PlayerSessionService) GetActivePlayerSessions(serverID uint) []models.PlayerSession {
	var sessions []models.PlayerSession
	p.db.Preload("Player").Where("server_id = ? AND leave_time IS NULL", serverID).Find(&sessions)
	return sessions
}

// ManualCleanupOldSessions 手动清理极长时间的旧会话（仅在确认需要时调用）
func (p *PlayerSessionService) ManualCleanupOldSessions(cutoffHours int) {
	if cutoffHours < 24 {
		log.Printf("安全起见，cutoffHours 必须大于等于24小时")
		return
	}
	
	cutoff := time.Now().Add(-time.Duration(cutoffHours) * time.Hour)
	
	var oldSessions []models.PlayerSession
	p.db.Where("leave_time IS NULL AND join_time < ?", cutoff).Find(&oldSessions)
	
	log.Printf("准备清理 %d 个超过 %d 小时的旧会话", len(oldSessions), cutoffHours)
	
	for _, session := range oldSessions {
		now := time.Now()
		duration := int(now.Sub(session.JoinTime).Seconds())
		
		session.LeaveTime = &now
		session.Duration = duration
		p.db.Save(&session)
		
		// 更新玩家总在线时间
		var player models.Player
		if p.db.First(&player, session.PlayerID).Error == nil {
			player.TotalPlaytime += duration
			player.LastSeen = now
			p.db.Save(&player)
		}
		
		log.Printf("清理旧会话: 玩家ID=%d, 服务器ID=%d, 持续时间=%d秒", 
			session.PlayerID, session.ServerID, duration)
	}
}