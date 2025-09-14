package db

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"etamonitor/internal/models"

	"gorm.io/gorm"
)

// OptimizationService 数据库优化服务
type OptimizationService struct {
	db     *gorm.DB
	dbPath string // 添加数据库文件路径
}

// NewOptimizationService 创建数据库优化服务
func NewOptimizationService(db *gorm.DB) *OptimizationService {
	return &OptimizationService{db: db, dbPath: ""}
}

// NewOptimizationServiceWithPath 创建带路径的数据库优化服务
func NewOptimizationServiceWithPath(db *gorm.DB, dbPath string) *OptimizationService {
	return &OptimizationService{db: db, dbPath: dbPath}
}

// OptimizeDatabase 执行数据库优化
func (s *OptimizationService) OptimizeDatabase() (*OptimizationResult, error) {
	log.Println("开始数据库优化...")

	result := &OptimizationResult{
		StartTime: time.Now(),
	}

	// 优化前自动创建备份
	if s.dbPath != "" {
		log.Println("正在创建优化前备份...")
		backupService := NewBackupService(s.db, s.dbPath)

		// 创建备份目录
		backupDir := filepath.Dir(s.dbPath) + "/backups"
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			log.Printf("警告: 无法创建备份目录: %v，跳过备份步骤", err)
		} else {
			backupResult, err := backupService.CreateBackup(backupDir)
			if err != nil {
				log.Printf("警告: 创建优化前备份失败: %v，继续优化过程", err)
			} else {
				log.Printf("优化前备份创建成功: %s", backupResult.BackupPath)
			}
		}
	}

	// 获取优化前的数据库文件大小
	preSize, err := s.getDatabaseFileSize()
	if err != nil {
		log.Printf("警告: 无法获取优化前数据库文件大小: %v", err)
	} else {
		result.PreDatabaseSize = preSize
	}

	// 获取优化前的统计信息
	if err := s.getPreOptimizationStats(result); err != nil {
		return nil, fmt.Errorf("获取优化前统计失败: %v", err)
	}

	// 执行服务器统计数据优化
	serverStatsDeleted, err := s.optimizeServerStats()
	if err != nil {
		return nil, fmt.Errorf("服务器统计数据优化失败: %v", err)
	}
	result.ServerStatsDeleted = serverStatsDeleted

	// 执行玩家活动数据优化
	activitiesDeleted, err := s.optimizePlayerActivities()
	if err != nil {
		return nil, fmt.Errorf("玩家活动数据优化失败: %v", err)
	}
	result.PlayerActivitiesDeleted = activitiesDeleted

	// 清理孤立的玩家会话数据
	sessionsDeleted, err := s.cleanupPlayerSessions()
	if err != nil {
		return nil, fmt.Errorf("玩家会话数据清理失败: %v", err)
	}
	result.PlayerSessionsDeleted = sessionsDeleted

	// 获取优化后的统计信息
	if err := s.getPostOptimizationStats(result); err != nil {
		return nil, fmt.Errorf("获取优化后统计失败: %v", err)
	}

	// 计算总删除记录数和估算节省空间
	result.DeletedRecords = serverStatsDeleted + activitiesDeleted + sessionsDeleted

	// 执行 VACUUM 操作来回收数据库空间
	log.Println("执行 VACUUM 操作回收数据库空间...")
	vacuumStart := time.Now()
	if err := s.db.Exec("VACUUM").Error; err != nil {
		log.Printf("警告: VACUUM 操作失败，但不影响数据优化结果: %v", err)
		// 估算每条记录平均占用 200 字节的空间
		result.SpaceSaved = result.DeletedRecords * 200
	} else {
		vacuumDuration := time.Since(vacuumStart)
		log.Printf("VACUUM 操作完成，耗时: %v", vacuumDuration)

		// 获取优化后的数据库文件大小
		postSize, err := s.getDatabaseFileSize()
		if err != nil {
			log.Printf("警告: 无法获取优化后数据库文件大小: %v", err)
			// 估算节省空间
			result.SpaceSaved = result.DeletedRecords * 200
		} else {
			result.PostDatabaseSize = postSize
			// 计算实际节省的空间
			if result.PreDatabaseSize > postSize {
				result.SpaceSaved = result.PreDatabaseSize - postSize
			} else {
				result.SpaceSaved = 0
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.PreDatabaseSize > 0 && result.PostDatabaseSize > 0 {
		log.Printf("数据库优化完成: 删除 %d 条服务器统计, %d 条玩家活动, %d 条玩家会话, 总计删除 %d 条记录",
			serverStatsDeleted, activitiesDeleted, sessionsDeleted, result.DeletedRecords)
		log.Printf("数据库文件大小: %d bytes -> %d bytes, 节省空间: %d bytes, 耗时: %v",
			result.PreDatabaseSize, result.PostDatabaseSize, result.SpaceSaved, result.Duration)
	} else {
		log.Printf("数据库优化完成: 删除 %d 条服务器统计, %d 条玩家活动, %d 条玩家会话, 总计删除 %d 条记录，估算节省空间 %d 字节, 耗时 %v",
			serverStatsDeleted, activitiesDeleted, sessionsDeleted, result.DeletedRecords, result.SpaceSaved, result.Duration)
	}

	return result, nil
}

// optimizeServerStats 优化服务器统计数据
// 使用智能算法保留关键信息：峰值、低谷、趋势变化点
func (s *OptimizationService) optimizeServerStats() (int64, error) {
	now := time.Now()
	totalDeleted := int64(0)

	var serverIDs []uint
	if err := s.db.Model(&models.Server{}).Pluck("id", &serverIDs).Error; err != nil {
		return 0, err
	}

	for _, serverID := range serverIDs {
		// 分时段智能优化
		deleted, err := s.smartOptimizeServerData(serverID, now)
		if err != nil {
			log.Printf("智能优化服务器 %d 数据失败: %v", serverID, err)
			continue
		}
		totalDeleted += deleted
	}

	log.Printf("服务器统计智能优化完成: 删除 %d 条记录", totalDeleted)
	return totalDeleted, nil
}

// smartOptimizeServerData 智能优化单个服务器的数据
func (s *OptimizationService) smartOptimizeServerData(serverID uint, now time.Time) (int64, error) {
	totalDeleted := int64(0)

	// 1. 删除超过1年的数据（保留长期历史）
	cutoff1Year := now.AddDate(-1, 0, 0)
	deleted, err := s.deleteOldData(serverID, cutoff1Year)
	if err != nil {
		return totalDeleted, err
	}
	totalDeleted += deleted

	// 2. 优化6个月-1年的数据：保留每日峰值和关键变化点
	cutoff6Month := now.AddDate(0, -6, 0)
	deleted, err = s.optimizeWithKeyPoints(serverID, cutoff1Year, cutoff6Month, 24*time.Hour, "daily")
	if err != nil {
		return totalDeleted, err
	}
	totalDeleted += deleted

	// 3. 优化1个月-6个月的数据：保留每小时的关键数据点
	cutoff1Month := now.AddDate(0, -1, 0)
	deleted, err = s.optimizeWithKeyPoints(serverID, cutoff6Month, cutoff1Month, time.Hour, "hourly")
	if err != nil {
		return totalDeleted, err
	}
	totalDeleted += deleted

	// 4. 优化1周-1个月的数据：保留每30分钟的关键数据点
	cutoff1Week := now.AddDate(0, 0, -7)
	deleted, err = s.optimizeWithKeyPoints(serverID, cutoff1Month, cutoff1Week, 30*time.Minute, "30min")
	if err != nil {
		return totalDeleted, err
	}
	totalDeleted += deleted

	// 5. 保留最近1周的所有数据（实时监控需要）

	return totalDeleted, nil
}

// optimizeWithKeyPoints 智能优化：保留关键数据点（峰值、低谷、趋势变化）
func (s *OptimizationService) optimizeWithKeyPoints(serverID uint, startTime, endTime time.Time, interval time.Duration, intervalType string) (int64, error) {
	// 先检查数据量，避免内存问题
	var count int64
	if err := s.db.Model(&models.ServerStat{}).Where("server_id = ? AND timestamp >= ? AND timestamp < ?",
		serverID, startTime, endTime).Count(&count).Error; err != nil {
		return 0, err
	}

	if count == 0 {
		return 0, nil
	}

	// 如果数据量超过100,000条，分批处理
	if count > 100000 {
		return s.optimizeLargeDataset(serverID, startTime, endTime, interval, intervalType)
	}

	var stats []models.ServerStat
	if err := s.db.Where("server_id = ? AND timestamp >= ? AND timestamp < ?",
		serverID, startTime, endTime).Order("timestamp ASC").Find(&stats).Error; err != nil {
		return 0, err
	}

	if len(stats) <= 10 { // 数据点太少，不优化
		return 0, nil
	}

	return s.processStatsOptimization(stats, serverID, intervalType)
}

// groupStatsByInterval 按时间间隔分组统计数据
func (s *OptimizationService) groupStatsByInterval(stats []models.ServerStat, interval time.Duration) [][]models.ServerStat {
	if len(stats) == 0 {
		return nil
	}

	var groups [][]models.ServerStat
	var currentGroup []models.ServerStat
	groupStartTime := stats[0].Timestamp

	for _, stat := range stats {
		if stat.Timestamp.Sub(groupStartTime) >= interval {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []models.ServerStat{stat}
			groupStartTime = stat.Timestamp
		} else {
			currentGroup = append(currentGroup, stat)
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// findKeyPointsInGroup 在数据组中找到关键数据点
func (s *OptimizationService) findKeyPointsInGroup(group []models.ServerStat) []models.ServerStat {
	if len(group) <= 3 {
		return group // 数据点太少，全部保留
	}

	var keyPoints []models.ServerStat

	// 保留组内的最大值和最小值
	maxStat := group[0]
	minStat := group[0]

	for _, stat := range group {
		if stat.PlayersOnline > maxStat.PlayersOnline {
			maxStat = stat
		}
		if stat.PlayersOnline < minStat.PlayersOnline {
			minStat = stat
		}
	}

	keyPoints = append(keyPoints, maxStat)
	if minStat.ID != maxStat.ID {
		keyPoints = append(keyPoints, minStat)
	}

	// 保留时间跨度中点的数据
	midIndex := len(group) / 2
	if group[midIndex].ID != maxStat.ID && group[midIndex].ID != minStat.ID {
		keyPoints = append(keyPoints, group[midIndex])
	}

	return keyPoints
}

// findGlobalKeyPoints 找到全局关键数据点（最高峰值和最低谷值）
func (s *OptimizationService) findGlobalKeyPoints(stats []models.ServerStat, maxPoints int) []models.ServerStat {
	if len(stats) <= maxPoints {
		return stats
	}

	// 创建数据点的副本并按玩家数排序
	type statWithIndex struct {
		stat models.ServerStat
		idx  int
	}

	var sortedStats []statWithIndex
	for i, stat := range stats {
		sortedStats = append(sortedStats, statWithIndex{stat, i})
	}

	// 分别获取最高的几个点和最低的几个点
	var keyPoints []models.ServerStat
	pointsAdded := make(map[uint]struct{})

	// 添加最高峰值点
	maxPointsHalf := maxPoints / 2
	for i := 0; i < len(sortedStats) && len(keyPoints) < maxPointsHalf; i++ {
		// 按玩家数降序排列后取前几个
		maxIdx := 0
		maxPlayers := -1
		for j, s := range sortedStats {
			if s.stat.PlayersOnline > maxPlayers {
				if _, exists := pointsAdded[s.stat.ID]; !exists {
					maxPlayers = s.stat.PlayersOnline
					maxIdx = j
				}
			}
		}
		if maxPlayers >= 0 {
			keyPoints = append(keyPoints, sortedStats[maxIdx].stat)
			pointsAdded[sortedStats[maxIdx].stat.ID] = struct{}{}
			sortedStats[maxIdx].stat.PlayersOnline = -1 // 标记为已使用
		}
	}

	// 添加最低谷值点
	for i := 0; i < len(stats) && len(keyPoints) < maxPointsHalf*2; i++ {
		minIdx := 0
		minPlayers := 999999
		for j, s := range stats {
			if s.PlayersOnline < minPlayers {
				if _, exists := pointsAdded[s.ID]; !exists {
					minPlayers = s.PlayersOnline
					minIdx = j
				}
			}
		}
		if minPlayers < 999999 {
			keyPoints = append(keyPoints, stats[minIdx])
			pointsAdded[stats[minIdx].ID] = struct{}{}
			stats[minIdx].PlayersOnline = 999999 // 标记为已使用
		}
	}

	return keyPoints
}

// findTrendChangePoints 识别趋势变化点
func (s *OptimizationService) findTrendChangePoints(stats []models.ServerStat) []models.ServerStat {
	if len(stats) < 5 {
		return nil
	}

	var changePoints []models.ServerStat

	// 使用滑动窗口检测趋势变化
	windowSize := 3
	for i := windowSize; i < len(stats)-windowSize; i++ {
		// 计算前窗口和后窗口的平均值
		var prevSum, nextSum int
		for j := i - windowSize; j < i; j++ {
			prevSum += stats[j].PlayersOnline
		}
		for j := i + 1; j <= i+windowSize; j++ {
			nextSum += stats[j].PlayersOnline
		}

		prevAvg := float64(prevSum) / float64(windowSize)
		nextAvg := float64(nextSum) / float64(windowSize)
		currentValue := float64(stats[i].PlayersOnline)

		// 检测显著的趋势变化（超过20%的变化）
		threshold := 0.2
		if (prevAvg > 0 && math.Abs(nextAvg-prevAvg)/prevAvg > threshold) ||
			(currentValue > 0 && (math.Abs(currentValue-prevAvg)/max(currentValue, prevAvg) > threshold ||
				math.Abs(nextAvg-currentValue)/max(nextAvg, currentValue) > threshold)) {
			changePoints = append(changePoints, stats[i])
		}
	}

	return changePoints
}

// deleteOldData 删除超过指定时间的旧数据
func (s *OptimizationService) deleteOldData(serverID uint, cutoff time.Time) (int64, error) {
	totalDeleted := int64(0)
	batchSize := 2000

	for {
		res := s.db.Where("server_id = ? AND timestamp < ?", serverID, cutoff).
			Limit(batchSize).Delete(&models.ServerStat{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}
		if res.RowsAffected == 0 {
			break
		}
		totalDeleted += res.RowsAffected
		time.Sleep(5 * time.Millisecond)
	}

	return totalDeleted, nil
}

// optimizePlayerActivities 优化玩家活动数据
func (s *OptimizationService) optimizePlayerActivities() (int64, error) {
	// 保留最近6个月的玩家活动记录（相比之前更加保守）
	cutoff := time.Now().AddDate(0, -6, 0)
	totalDeleted := int64(0)

	// 分批删除以避免长时间锁定数据库
	batchSize := 2000
	for {
		res := s.db.Where("timestamp < ?", cutoff).Limit(batchSize).Delete(&models.PlayerActivity{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}

		if res.RowsAffected == 0 {
			break
		}

		totalDeleted += res.RowsAffected
		time.Sleep(5 * time.Millisecond)
	}

	log.Printf("删除6个月前的玩家活动记录: %d 条", totalDeleted)
	return totalDeleted, nil
}

// cleanupPlayerSessions 清理孤立的玩家会话数据
func (s *OptimizationService) cleanupPlayerSessions() (int64, error) {
	// 保留最近1年的已完成会话（更加保守的策略）
	cutoff := time.Now().AddDate(-1, 0, 0)
	totalDeleted := int64(0)

	// 分批删除已完成的会话
	batchSize := 2000
	for {
		res := s.db.Where("leave_time IS NOT NULL AND leave_time < ?", cutoff).Limit(batchSize).Delete(&models.PlayerSession{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}

		if res.RowsAffected == 0 {
			break
		}

		totalDeleted += res.RowsAffected
		time.Sleep(5 * time.Millisecond)
	}

	log.Printf("删除1年前的完成会话: %d 条", totalDeleted)
	return totalDeleted, nil
}

// getPreOptimizationStats 获取优化前的统计信息
func (s *OptimizationService) getPreOptimizationStats(result *OptimizationResult) error {
	var err error

	err = s.db.Model(&models.ServerStat{}).Count(&result.PreStats.ServerStats).Error
	if err != nil {
		return err
	}

	err = s.db.Model(&models.PlayerActivity{}).Count(&result.PreStats.PlayerActivities).Error
	if err != nil {
		return err
	}

	err = s.db.Model(&models.PlayerSession{}).Count(&result.PreStats.PlayerSessions).Error
	if err != nil {
		return err
	}

	return nil
}

// getPostOptimizationStats 获取优化后的统计信息
func (s *OptimizationService) getPostOptimizationStats(result *OptimizationResult) error {
	var err error

	err = s.db.Model(&models.ServerStat{}).Count(&result.PostStats.ServerStats).Error
	if err != nil {
		return err
	}

	err = s.db.Model(&models.PlayerActivity{}).Count(&result.PostStats.PlayerActivities).Error
	if err != nil {
		return err
	}

	err = s.db.Model(&models.PlayerSession{}).Count(&result.PostStats.PlayerSessions).Error
	if err != nil {
		return err
	}

	return nil
}

// GetDatabaseStats 获取数据库统计信息
func (s *OptimizationService) GetDatabaseStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// 获取各表的记录数
	s.db.Model(&models.Server{}).Count(&stats.Servers)
	s.db.Model(&models.ServerStat{}).Count(&stats.ServerStats)
	s.db.Model(&models.Player{}).Count(&stats.Players)
	s.db.Model(&models.PlayerSession{}).Count(&stats.PlayerSessions)
	s.db.Model(&models.PlayerActivity{}).Count(&stats.PlayerActivities)
	s.db.Model(&models.PlayerTitle{}).Count(&stats.PlayerTitles)
	s.db.Model(&models.User{}).Count(&stats.Users)

	// 计算数据时间范围
	var oldestStat, newestStat models.ServerStat
	s.db.Order("timestamp ASC").First(&oldestStat)
	s.db.Order("timestamp DESC").First(&newestStat)

	if !oldestStat.Timestamp.IsZero() {
		stats.OldestData = &oldestStat.Timestamp
	}
	if !newestStat.Timestamp.IsZero() {
		stats.NewestData = &newestStat.Timestamp
	}

	return stats, nil
}

// getDatabaseFileSize 获取数据库文件大小
func (s *OptimizationService) getDatabaseFileSize() (int64, error) {
	if s.dbPath == "" {
		return 0, fmt.Errorf("数据库文件路径未设置")
	}

	info, err := os.Stat(s.dbPath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	StartTime               time.Time     `json:"start_time"`
	EndTime                 time.Time     `json:"end_time"`
	Duration                time.Duration `json:"duration"`
	PreStats                DatabaseStats `json:"pre_stats"`
	PostStats               DatabaseStats `json:"post_stats"`
	ServerStatsDeleted      int64         `json:"server_stats_deleted"`
	PlayerActivitiesDeleted int64         `json:"player_activities_deleted"`
	PlayerSessionsDeleted   int64         `json:"player_sessions_deleted"`
	DeletedRecords          int64         `json:"deleted_records"`    // 总删除记录数
	SpaceSaved              int64         `json:"space_saved"`        // 实际节省的空间
	PreDatabaseSize         int64         `json:"pre_database_size"`  // 优化前数据库文件大小
	PostDatabaseSize        int64         `json:"post_database_size"` // 优化后数据库文件大小
}

// DatabaseStats 数据库统计信息
type DatabaseStats struct {
	Servers          int64      `json:"servers"`
	ServerStats      int64      `json:"server_stats"`
	Players          int64      `json:"players"`
	PlayerSessions   int64      `json:"player_sessions"`
	PlayerActivities int64      `json:"player_activities"`
	PlayerTitles     int64      `json:"player_titles"`
	Users            int64      `json:"users"`
	OldestData       *time.Time `json:"oldest_data,omitempty"`
	NewestData       *time.Time `json:"newest_data,omitempty"`
}

// max 返回两个float64中的较大值
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// processStatsOptimization 处理统计数据优化
func (s *OptimizationService) processStatsOptimization(stats []models.ServerStat, serverID uint, intervalType string) (int64, error) {
	keepIDs := make(map[uint]struct{})

	// 1. 始终保留时间段的首尾数据点
	keepIDs[stats[0].ID] = struct{}{}
	keepIDs[stats[len(stats)-1].ID] = struct{}{}

	// 2. 按时间间隔分组，保留每组的关键数据点
	var interval time.Duration
	switch intervalType {
	case "daily":
		interval = 24 * time.Hour
	case "hourly":
		interval = time.Hour
	case "30min":
		interval = 30 * time.Minute
	default:
		interval = time.Hour
	}

	groupedStats := s.groupStatsByInterval(stats, interval)
	for _, group := range groupedStats {
		keyPoints := s.findKeyPointsInGroup(group)
		for _, point := range keyPoints {
			keepIDs[point.ID] = struct{}{}
		}
	}

	// 3. 识别全局峰值和低谷（创建副本避免修改原数据）
	statsCopy := make([]models.ServerStat, len(stats))
	copy(statsCopy, stats)
	globalKeyPoints := s.findGlobalKeyPointsSafe(statsCopy, 10)
	for _, point := range globalKeyPoints {
		keepIDs[point.ID] = struct{}{}
	}

	// 4. 识别趋势变化点
	trendPoints := s.findTrendChangePointsSafe(stats)
	for _, point := range trendPoints {
		keepIDs[point.ID] = struct{}{}
	}

	// 计算需要删除的数据点
	var deleteIDs []uint
	for _, stat := range stats {
		if _, keep := keepIDs[stat.ID]; !keep {
			deleteIDs = append(deleteIDs, stat.ID)
		}
	}

	if len(deleteIDs) == 0 {
		return 0, nil
	}

	return s.batchDelete(deleteIDs, serverID, intervalType, len(keepIDs), len(stats))
}

// optimizeLargeDataset 处理大数据集
func (s *OptimizationService) optimizeLargeDataset(serverID uint, startTime, endTime time.Time, interval time.Duration, intervalType string) (int64, error) {
	log.Printf("服务器 %d [%s] 检测到大数据集，使用分批处理", serverID, intervalType)

	totalDeleted := int64(0)
	batchSize := 10000

	for offset := 0; ; offset += batchSize {
		var stats []models.ServerStat
		err := s.db.Where("server_id = ? AND timestamp >= ? AND timestamp < ?",
			serverID, startTime, endTime).
			Order("timestamp ASC").
			Offset(offset).
			Limit(batchSize).
			Find(&stats).Error

		if err != nil {
			return totalDeleted, err
		}

		if len(stats) == 0 {
			break
		}

		deleted, err := s.simpleIntervalSampling(stats, interval)
		if err != nil {
			return totalDeleted, err
		}

		totalDeleted += deleted
		time.Sleep(50 * time.Millisecond)
	}

	return totalDeleted, nil
}

// simpleIntervalSampling 简单间隔采样
func (s *OptimizationService) simpleIntervalSampling(stats []models.ServerStat, interval time.Duration) (int64, error) {
	if len(stats) <= 2 {
		return 0, nil
	}

	keepIDs := make(map[uint]struct{})
	keepIDs[stats[0].ID] = struct{}{}
	keepIDs[stats[len(stats)-1].ID] = struct{}{}

	lastKeptTime := stats[0].Timestamp
	for _, stat := range stats[1 : len(stats)-1] {
		if stat.Timestamp.Sub(lastKeptTime) >= interval {
			keepIDs[stat.ID] = struct{}{}
			lastKeptTime = stat.Timestamp
		}
	}

	var deleteIDs []uint
	for _, stat := range stats {
		if _, keep := keepIDs[stat.ID]; !keep {
			deleteIDs = append(deleteIDs, stat.ID)
		}
	}

	if len(deleteIDs) == 0 {
		return 0, nil
	}

	totalDeleted := int64(0)
	batchSize := 1000
	for i := 0; i < len(deleteIDs); i += batchSize {
		end := min(i+batchSize, len(deleteIDs))
		batch := deleteIDs[i:end]

		res := s.db.Where("id IN ?", batch).Delete(&models.ServerStat{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}
		totalDeleted += res.RowsAffected
		time.Sleep(5 * time.Millisecond)
	}

	return totalDeleted, nil
}

// batchDelete 批量删除操作
func (s *OptimizationService) batchDelete(deleteIDs []uint, serverID uint, intervalType string, keepCount, totalCount int) (int64, error) {
	totalDeleted := int64(0)
	batchSize := 1000

	for i := 0; i < len(deleteIDs); i += batchSize {
		end := min(i+batchSize, len(deleteIDs))
		batch := deleteIDs[i:end]

		res := s.db.Where("id IN ?", batch).Delete(&models.ServerStat{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}
		totalDeleted += res.RowsAffected
		time.Sleep(2 * time.Millisecond)
	}

	if totalDeleted > 0 {
		compressionRatio := float64(keepCount) / float64(totalCount) * 100
		log.Printf("服务器 %d [%s] 智能优化: 保留 %d/%d 条关键数据点 (%.1f%%), 删除 %d 条",
			serverID, intervalType, keepCount, totalCount, compressionRatio, totalDeleted)
	}

	return totalDeleted, nil
}

// findGlobalKeyPointsSafe 安全的全局关键点查找（不修改原数据）
func (s *OptimizationService) findGlobalKeyPointsSafe(stats []models.ServerStat, maxPoints int) []models.ServerStat {
	if len(stats) <= maxPoints {
		return stats
	}

	var keyPoints []models.ServerStat
	usedIndices := make(map[int]bool)

	// 找最高的几个点
	maxPointsHalf := maxPoints / 2
	for len(keyPoints) < maxPointsHalf {
		maxIdx := -1
		maxPlayers := -1
		for i, stat := range stats {
			if !usedIndices[i] && stat.PlayersOnline > maxPlayers {
				maxPlayers = stat.PlayersOnline
				maxIdx = i
			}
		}
		if maxIdx == -1 {
			break
		}
		keyPoints = append(keyPoints, stats[maxIdx])
		usedIndices[maxIdx] = true
	}

	// 找最低的几个点
	for len(keyPoints) < maxPoints {
		minIdx := -1
		minPlayers := 999999
		for i, stat := range stats {
			if !usedIndices[i] && stat.PlayersOnline < minPlayers {
				minPlayers = stat.PlayersOnline
				minIdx = i
			}
		}
		if minIdx == -1 {
			break
		}
		keyPoints = append(keyPoints, stats[minIdx])
		usedIndices[minIdx] = true
	}

	return keyPoints
}

// findTrendChangePointsSafe 识别趋势变化点（优化版）
func (s *OptimizationService) findTrendChangePointsSafe(stats []models.ServerStat) []models.ServerStat {
	if len(stats) < 7 {
		return nil
	}

	var changePoints []models.ServerStat
	windowSize := 3
	threshold := 0.3

	for i := windowSize; i < len(stats)-windowSize; i++ {
		var prevSum, nextSum int
		for j := i - windowSize; j < i; j++ {
			prevSum += stats[j].PlayersOnline
		}
		for j := i + 1; j <= i+windowSize; j++ {
			nextSum += stats[j].PlayersOnline
		}

		prevAvg := float64(prevSum) / float64(windowSize)
		nextAvg := float64(nextSum) / float64(windowSize)

		if prevAvg > 1 && nextAvg > 1 {
			if math.Abs(nextAvg-prevAvg)/max(prevAvg, nextAvg) > threshold {
				changePoints = append(changePoints, stats[i])
			}
		}
	}

	if len(changePoints) > 20 {
		changePoints = changePoints[:20]
	}

	return changePoints
}
