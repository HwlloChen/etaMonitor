package db

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// BackupService 数据库备份服务
type BackupService struct {
	db     *gorm.DB
	dbPath string
}

// NewBackupService 创建数据库备份服务
func NewBackupService(db *gorm.DB, dbPath string) *BackupService {
	return &BackupService{
		db:     db,
		dbPath: dbPath,
	}
}

// CreateBackup 创建数据库备份
func (s *BackupService) CreateBackup(backupDir string) (*BackupResult, error) {
	log.Println("开始创建数据库备份...")
	
	result := &BackupResult{
		StartTime: time.Now(),
	}
	
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %v", err)
	}
	
	// 生成备份文件名
	timestamp := result.StartTime.Format("20060102_150405")
	backupFileName := fmt.Sprintf("etamonitor_backup_%s.zip", timestamp)
	backupFilePath := filepath.Join(backupDir, backupFileName)
	
	// 获取数据库文件信息
	dbInfo, err := os.Stat(s.dbPath)
	if err != nil {
		return nil, fmt.Errorf("获取数据库文件信息失败: %v", err)
	}
	result.OriginalSize = dbInfo.Size()
	
	// 创建ZIP备份文件
	zipFile, err := os.Create(backupFilePath)
	if err != nil {
		return nil, fmt.Errorf("创建备份文件失败: %v", err)
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	// 添加数据库文件到ZIP
	if err := s.addFileToZip(zipWriter, s.dbPath, filepath.Base(s.dbPath)); err != nil {
		return nil, fmt.Errorf("添加数据库文件到备份失败: %v", err)
	}
	
	// 创建备份元数据
	metadata := s.createBackupMetadata()
	if err := s.addMetadataToZip(zipWriter, metadata); err != nil {
		return nil, fmt.Errorf("添加元数据到备份失败: %v", err)
	}
	
	// 关闭ZIP写入器以确保所有数据写入
	zipWriter.Close()
	zipFile.Close()
	
	// 获取备份文件信息
	backupInfo, err := os.Stat(backupFilePath)
	if err != nil {
		return nil, fmt.Errorf("获取备份文件信息失败: %v", err)
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.BackupPath = backupFilePath
	result.BackupSize = backupInfo.Size()
	result.CompressionRatio = float64(result.BackupSize) / float64(result.OriginalSize)
	
	log.Printf("数据库备份完成: %s (原大小: %d bytes, 备份大小: %d bytes, 压缩率: %.2f%%, 耗时: %v)", 
		backupFilePath, result.OriginalSize, result.BackupSize, result.CompressionRatio*100, result.Duration)
	
	return result, nil
}

// addFileToZip 添加文件到ZIP
func (s *BackupService) addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	zipFile, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}
	
	_, err = io.Copy(zipFile, file)
	return err
}

// addMetadataToZip 添加元数据到ZIP
func (s *BackupService) addMetadataToZip(zipWriter *zip.Writer, metadata *BackupMetadata) error {
	metadataFile, err := zipWriter.Create("backup_metadata.json")
	if err != nil {
		return err
	}
	
	metadataJSON := fmt.Sprintf(`{
  "backup_time": "%s",
  "version": "%s",
  "database_stats": {
    "servers": %d,
    "server_stats": %d,
    "players": %d,
    "player_sessions": %d,
    "player_activities": %d,
    "player_titles": %d,
    "users": %d
  },
  "oldest_data": "%s",
  "newest_data": "%s"
}`,
		metadata.BackupTime.Format(time.RFC3339),
		metadata.Version,
		metadata.DatabaseStats.Servers,
		metadata.DatabaseStats.ServerStats,
		metadata.DatabaseStats.Players,
		metadata.DatabaseStats.PlayerSessions,
		metadata.DatabaseStats.PlayerActivities,
		metadata.DatabaseStats.PlayerTitles,
		metadata.DatabaseStats.Users,
		formatTimePtr(metadata.OldestData),
		formatTimePtr(metadata.NewestData),
	)
	
	_, err = metadataFile.Write([]byte(metadataJSON))
	return err
}

// createBackupMetadata 创建备份元数据
func (s *BackupService) createBackupMetadata() *BackupMetadata {
	metadata := &BackupMetadata{
		BackupTime: time.Now(),
		Version:    "1.0.4-beta1", // 从配置中获取
	}
	
	// 获取数据库统计信息
	optimizationService := NewOptimizationService(s.db)
	stats, err := optimizationService.GetDatabaseStats()
	if err == nil {
		metadata.DatabaseStats = *stats
		metadata.OldestData = stats.OldestData
		metadata.NewestData = stats.NewestData
	}
	
	return metadata
}

// ListBackups 列出备份文件
func (s *BackupService) ListBackups(backupDir string) ([]BackupInfo, error) {
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %v", err)
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %v", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".zip" {
			continue
		}

		filePath := filepath.Join(backupDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		backup := BackupInfo{
			Name:         file.Name(),
			Path:         filePath,
			Size:         fileInfo.Size(),
			CreatedTime:  fileInfo.ModTime(),
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

// DeleteBackup 删除备份文件
func (s *BackupService) DeleteBackup(backupPath string) error {
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("删除备份文件失败: %v", err)
	}
	
	log.Printf("删除备份文件: %s", backupPath)
	return nil
}

// CleanupOldBackups 清理旧备份文件
func (s *BackupService) CleanupOldBackups(backupDir string, keepDays int) (*CleanupResult, error) {
	log.Printf("开始清理%d天前的备份文件...", keepDays)
	
	result := &CleanupResult{
		StartTime: time.Now(),
	}
	
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	backups, err := s.ListBackups(backupDir)
	if err != nil {
		return nil, err
	}
	
	var deletedFiles []string
	var totalSize int64
	
	for _, backup := range backups {
		if backup.CreatedTime.Before(cutoff) {
			if err := s.DeleteBackup(backup.Path); err != nil {
				log.Printf("删除备份文件失败: %s, 错误: %v", backup.Path, err)
				continue
			}
			deletedFiles = append(deletedFiles, backup.Name)
			totalSize += backup.Size
		}
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.DeletedFiles = deletedFiles
	result.DeletedCount = len(deletedFiles)
	result.SpaceFreed = totalSize
	
	log.Printf("清理完成: 删除 %d 个备份文件, 释放空间 %d bytes, 耗时 %v", 
		result.DeletedCount, result.SpaceFreed, result.Duration)
	
	return result, nil
}

// RestoreBackup 恢复数据库备份
func (s *BackupService) RestoreBackup(backupPath string) (*RestoreResult, error) {
	log.Printf("开始恢复数据库备份: %s", backupPath)
	
	result := &RestoreResult{
		StartTime:  time.Now(),
		BackupPath: backupPath,
	}
	
	// 验证备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	
	// 创建临时目录解压备份文件
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("etamonitor_restore_%d", time.Now().Unix()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录
	
	// 解压备份文件
	if err := s.extractBackup(backupPath, tempDir); err != nil {
		return nil, fmt.Errorf("解压备份文件失败: %v", err)
	}
	
	// 验证备份文件完整性
	dbFileName := filepath.Base(s.dbPath)
	extractedDBPath := filepath.Join(tempDir, dbFileName)
	if _, err := os.Stat(extractedDBPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("备份文件中缺少数据库文件")
	}
	
	// 读取备份元数据（如果存在）
	metadataPath := filepath.Join(tempDir, "backup_metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		metadata, err := s.readBackupMetadata(metadataPath)
		if err == nil {
			result.BackupMetadata = metadata
		}
	}
	
	// 检查数据库连接状态（但不关闭连接，避免影响运行中的应用）
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %v", err)
	}

	// 检查连接是否正常（不关闭）
	if err := sqlDB.Ping(); err != nil {
		log.Printf("警告: 数据库连接异常，但继续恢复过程: %v", err)
	}

	// 使用完整的备份流程来备份当前数据库
	log.Println("创建恢复前的完整备份...")
	backupDir := filepath.Dir(s.dbPath) + "/backups"
	currentBackupResult, err := s.CreateBackup(backupDir)
	if err != nil {
		return nil, fmt.Errorf("创建恢复前备份失败: %v", err)
	}
	result.CurrentDBBackupPath = currentBackupResult.BackupPath
	result.CurrentDBBackupResult = currentBackupResult
	log.Printf("恢复前备份创建成功: %s (原大小: %d bytes, 备份大小: %d bytes)",
		currentBackupResult.BackupPath, currentBackupResult.OriginalSize, currentBackupResult.BackupSize)

	// 替换数据库文件
	if err := s.copyFile(extractedDBPath, s.dbPath); err != nil {
		// 恢复失败，尝试还原原数据库
		log.Printf("数据库文件替换失败，尝试从备份恢复: %s", err)

		// 从刚才创建的完整备份中提取数据库文件进行还原
		tempRestoreDir := filepath.Join(os.TempDir(), fmt.Sprintf("etamonitor_emergency_restore_%d", time.Now().Unix()))
		if extractErr := s.extractBackup(currentBackupResult.BackupPath, tempRestoreDir); extractErr == nil {
			dbFileName := filepath.Base(s.dbPath)
			emergencyDBPath := filepath.Join(tempRestoreDir, dbFileName)
			if restoreErr := s.copyFile(emergencyDBPath, s.dbPath); restoreErr != nil {
				log.Printf("紧急恢复也失败: %v", restoreErr)
			} else {
				log.Println("紧急恢复成功，数据库已还原到恢复前状态")
			}
			os.RemoveAll(tempRestoreDir) // 清理临时目录
		}

		return nil, fmt.Errorf("替换数据库文件失败: %v", err)
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	log.Printf("数据库恢复完成: %s (耗时: %v)", backupPath, result.Duration)
	log.Printf("重要提示: 数据库文件已被替换，强烈建议重启应用程序以确保数据库连接正常工作")

	return result, nil
}

// extractBackup 解压备份文件
func (s *BackupService) extractBackup(backupPath, destDir string) error {
	zipFile, err := zip.OpenReader(backupPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	
	for _, file := range zipFile.File {
		destPath := filepath.Join(destDir, file.Name)
		
		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		
		// 打开压缩文件中的文件
		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()
		
		// 创建目标文件
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()
		
		// 复制文件内容
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}
	}
	
	return nil
}

// copyFile 复制文件
func (s *BackupService) copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, srcFile)
	return err
}

// readBackupMetadata 读取备份元数据
func (s *BackupService) readBackupMetadata(metadataPath string) (*BackupMetadata, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}
	
	// 这里应该使用JSON解析，为简化起见，返回基本信息
	metadata := &BackupMetadata{
		BackupTime: time.Now(), // 实际应该从JSON解析
		Version:    "unknown",
	}
	
	// TODO: 实现完整的JSON解析
	_ = data
	
	return metadata, nil
}

// ValidateBackup 验证备份文件
func (s *BackupService) ValidateBackup(backupPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		BackupPath: backupPath,
		Valid:      false,
	}
	
	// 检查文件是否存在
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		result.Errors = append(result.Errors, "备份文件不存在")
		return result, nil
	}
	
	result.FileSize = fileInfo.Size()
	
	// 检查是否是ZIP文件
	if filepath.Ext(backupPath) != ".zip" {
		result.Errors = append(result.Errors, "不是有效的ZIP文件")
		return result, nil
	}
	
	// 尝试打开ZIP文件
	zipReader, err := zip.OpenReader(backupPath)
	if err != nil {
		result.Errors = append(result.Errors, "无法打开ZIP文件: "+err.Error())
		return result, nil
	}
	defer zipReader.Close()
	
	// 检查必需的文件
	hasDBFile := false
	hasMetadata := false
	
	for _, file := range zipReader.File {
		if file.Name == filepath.Base(s.dbPath) {
			hasDBFile = true
		}
		if file.Name == "backup_metadata.json" {
			hasMetadata = true
		}
	}
	
	if !hasDBFile {
		result.Errors = append(result.Errors, "备份文件中缺少数据库文件")
	}
	
	if hasMetadata {
		result.HasMetadata = true
	}
	
	// 如果没有错误，标记为有效
	if len(result.Errors) == 0 {
		result.Valid = true
	}
	
	return result, nil
}

// BackupResult 备份结果
type BackupResult struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	BackupPath       string        `json:"backup_path"`
	OriginalSize     int64         `json:"original_size"`
	BackupSize       int64         `json:"backup_size"`
	CompressionRatio float64       `json:"compression_ratio"`
}

// BackupInfo 备份信息
type BackupInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedTime time.Time `json:"created_time"`
}

// BackupMetadata 备份元数据
type BackupMetadata struct {
	BackupTime     time.Time     `json:"backup_time"`
	Version        string        `json:"version"`
	DatabaseStats  DatabaseStats `json:"database_stats"`
	OldestData     *time.Time    `json:"oldest_data,omitempty"`
	NewestData     *time.Time    `json:"newest_data,omitempty"`
}

// CleanupResult 清理结果
type CleanupResult struct {
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	DeletedFiles  []string      `json:"deleted_files"`
	DeletedCount  int           `json:"deleted_count"`
	SpaceFreed    int64         `json:"space_freed"`
}

// RestoreResult 恢复结果
type RestoreResult struct {
	StartTime             time.Time        `json:"start_time"`
	EndTime               time.Time        `json:"end_time"`
	Duration              time.Duration    `json:"duration"`
	BackupPath            string           `json:"backup_path"`
	Success               bool             `json:"success"`
	BackupMetadata        *BackupMetadata  `json:"backup_metadata,omitempty"`
	CurrentDBBackupPath   string           `json:"current_db_backup_path"`
	CurrentDBBackupResult *BackupResult    `json:"current_db_backup_result,omitempty"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	BackupPath  string   `json:"backup_path"`
	Valid       bool     `json:"valid"`
	FileSize    int64    `json:"file_size"`
	HasMetadata bool     `json:"has_metadata"`
	Errors      []string `json:"errors"`
}

// formatTimePtr 格式化时间指针
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}