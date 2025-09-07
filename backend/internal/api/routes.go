package api

import (
	"io"
	"strings"
	"time"

	"etamonitor/internal/auth"
	"etamonitor/internal/config"
	"etamonitor/internal/static"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// 中间件
	router.Use(auth.CORSMiddleware())

	// API组
	api := router.Group("/api")

	// 公开路由
	public := api.Group("/")
	setupPublicRoutes(public, db, cfg)

	// 需要认证的路由
	protected := api.Group("/")
	protected.Use(auth.AuthMiddleware(cfg.JWTSecret))
	setupProtectedRoutes(protected, db, cfg)

	// WebSocket路由
	router.GET("/ws", handleWebSocket)

	// WebSocket统计API (需要认证)
	api.GET("/websocket/stats", auth.AuthMiddleware(cfg.JWTSecret), handleWebSocketStats)

	// 静态文件服务，自动处理 MIME 类型，只映射 /assets
	router.StaticFS("/assets", static.GetAssetsFileSystem())

	// SPA路由处理
	router.NoRoute(func(c *gin.Context) {
		// API 和 WebSocket 请求不处理
		if strings.HasPrefix(c.Request.URL.Path, "/api") ||
			strings.HasPrefix(c.Request.URL.Path, "/ws") {
			c.Next()
			return
		}

		// 打开 index.html
		file, err := static.GetFileSystem().Open("index.html")
		if err != nil {
			c.String(500, "Internal Server Error")
			return
		}
		defer file.Close()

		// 读取index.html内容
		html, err := io.ReadAll(file)
		if err != nil {
			c.String(500, "Internal Server Error")
			return
		}

		// 设置正确的Content-Type
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(html))
	})
}

func setupPublicRoutes(r *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	// 创建登录限流器：每分钟最多10次请求
	loginLimiter := auth.NewRateLimiter(10, time.Minute)

	// 认证
	auth := r.Group("/auth")
	{
		auth.POST("/login", loginLimiter.Middleware(), handleLogin(db, cfg.JWTSecret, cfg.JWTExpiresIn))
		auth.POST("/refresh", handleRefresh(cfg.JWTSecret, cfg.JWTExpiresIn))
	}

	// 服务器信息（只读）
	servers := r.Group("/servers")
	{
		servers.GET("/", handleGetServers(db))
		servers.GET("/:id", handleGetServer(db))
		servers.GET("/:id/players", handleGetServerOnlinePlayers(db))
	}

	// 统计数据
	stats := r.Group("/stats")
	{
		stats.GET("/overview", handleStatsOverview(db))
		stats.GET("/servers/:id", handleServerStats(db))
		stats.GET("/players/:id", handlePlayerStats(db))
	}

	// 玩家信息
	players := r.Group("/players")
	{
		players.GET("/", handleGetPlayers(db))
		players.GET("/:id", handleGetPlayer(db))
		players.GET("/:id/sessions", handleGetPlayerSessions(db))
	}
	
	// 活动记录
	activities := r.Group("/activities")
	{
		activities.GET("/recent", handleGetRecentActivities(db, cfg))
	}
}

func setupProtectedRoutes(r *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	// 认证相关
	auth := r.Group("/auth")
	{
		auth.POST("/logout", handleLogout())
		auth.GET("/me", handleMe())
		auth.POST("/change-password", handleChangePassword(db))
	}

	// 服务器管理
	servers := r.Group("/servers")
	{
		servers.POST("/", handleCreateServer(db))
		servers.PUT("/:id", handleUpdateServer(db))
		servers.DELETE("/:id", handleDeleteServer(db))
		servers.POST("/:id/ping", handlePingServer(db))
		servers.POST("/detect", handleDetectServer())
	}

	// 用户管理
	users := r.Group("/users")
	{
		users.GET("/", handleGetUsers(db))
		users.POST("/", handleCreateUser(db))
		users.PUT("/:id", handleUpdateUser(db))
		users.DELETE("/:id", handleDeleteUser(db))
	}
}
