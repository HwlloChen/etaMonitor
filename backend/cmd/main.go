package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"etamonitor/internal/api"
	"etamonitor/internal/cli"
	"etamonitor/internal/config"
	"etamonitor/internal/db"
	"etamonitor/internal/monitor"
	"etamonitor/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// 打印版本信息
	log.Printf("=== etaMonitor %s ===\n", config.Version)
	log.Printf("Build Time: %s\n", config.BuildTime)
	log.Printf("Git Commit: %s\n", config.GitCommit)
	log.Println("========================")

	// 解析命令行参数
	setadmin := flag.Bool("setadmin", false, "设置管理员账户")
	configPath := flag.String("c", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg := config.Load(*configPath)

	// 初始化数据库
	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 如果是设置管理员模式
	if *setadmin {
		if err := cli.SetupAdmin(database, false); err != nil {
			log.Fatal("Failed to setup admin account:", err)
		}
		os.Exit(0)
	}

	// 初始化WebSocket
	websocket.InitWebSocket()
	log.Println("WebSocket initialized")

	// 设置Gin模式（release/debug/test）
	gin.SetMode(cfg.Environment)

	// 创建Gin路由器（不带默认 Logger/Recovery）
	router := gin.New()
	// 如需日志和 panic 恢复，可手动添加
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 设置受信任的代理，以修复"You trusted all proxies, this is NOT safe"警告
	// 包括本地网络和常见的反向代理IP范围
	err = router.SetTrustedProxies([]string{
		"127.0.0.1",      // localhost
		"10.0.0.0/8",     // 私有网络 10.x.x.x
		"172.16.0.0/12",  // 私有网络 172.16-31.x.x
		"192.168.0.0/16", // 私有网络 192.168.x.x
	})
	if err != nil {
		log.Printf("Warning: Failed to set trusted proxies: %v", err)
	}

	// 初始化API路由
	api.SetupRoutes(router, database, cfg)

	// 启动服务器监控
	monitorService := monitor.NewService(database, cfg)
	go monitorService.Start()

	// 启动服务器
	address := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Printf("Server starting on %s", address)
	if err := router.Run(address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
