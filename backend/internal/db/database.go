package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"etamonitor/internal/cli"
	"etamonitor/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(dbPath string) (*gorm.DB, error) {
	// 确保数据库目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移数据库表
	err = db.AutoMigrate(
		&models.Server{},
		&models.ServerStat{},
		&models.Player{},
		&models.PlayerSession{},
		&models.PlayerActivity{},
		&models.PlayerTitle{},
		&models.User{},
	)
	if err != nil {
		return nil, err
	}

	// 创建默认管理员用户
	if err := createDefaultAdmin(db); err != nil {
		log.Printf("Error: Failed to create admin user: %v", err)
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	return db, nil
}

// createDefaultAdmin 创建默认管理员用户
func createDefaultAdmin(db *gorm.DB) error {
	// 检查是否已有管理员用户
	var count int64
	db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

	if count > 0 {
		// log.Println("Admin user already exists, skipping creation")
		return nil
	}

	// 首次启动时交互式设置管理员账户
	if err := cli.SetupAdmin(db, true); err != nil {
		return fmt.Errorf("failed to setup admin account: %w", err)
	}

	return nil
}