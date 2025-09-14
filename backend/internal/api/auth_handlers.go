package api

import (
	"net/http"
	"time"

	"etamonitor/internal/auth"
	"etamonitor/internal/models"
	"etamonitor/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// =================================================================================
// Authenticated Handlers (需要认证)
//
// 此文件包含所有需要用户认证 (JWT Token) 才能访问的API处理器。
// 主要用于后台管理、服务器增删改、敏感数据统计等操作。
// =================================================================================

// handleLogin 登录处理
func handleLogin(db *gorm.DB, jwtSecret string, jwtExpiresIn time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   map[string]interface{}{"code": "VALIDATION_ERROR", "message": "请求参数验证失败", "details": err.Error()},
			})
			return
		}

		var user models.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   map[string]interface{}{"code": "INVALID_CREDENTIALS", "message": "用户名或密码错误"},
			})
			return
		}

		if err := auth.VerifyPassword(req.Password, user.Password); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   map[string]interface{}{"code": "INVALID_CREDENTIALS", "message": "用户名或密码错误"},
			})
			return
		}

		now := time.Now()
		db.Model(&user).Update("last_login_at", &now)

		token, err := auth.GenerateToken(user.ID, user.Username, user.Role, jwtSecret, jwtExpiresIn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   map[string]interface{}{"code": "INTERNAL_ERROR", "message": "生成token失败"},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"token":      token,
			"expires_in": int(jwtExpiresIn.Seconds()),
			"user":       gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
		})
	}
}

// handleRefresh 刷新token
func handleRefresh(jwtSecret string, jwtExpiresIn time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "未登录或登录已过期",
				},
			})
			return
		}

		username, _ := c.Get("username")
		role, _ := c.Get("role")

		// 类型断言时进行安全检查
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "用户ID类型错误",
				},
			})
			return
		}

		uname, ok := username.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "用户名类型错误",
				},
			})
			return
		}

		r, ok := role.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "用户角色类型错误",
				},
			})
			return
		}

		token, err := auth.GenerateToken(uid, uname, r, jwtSecret, jwtExpiresIn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOKEN_GENERATION_FAILED",
					"message": "生成token失败",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"token":      token,
			"expires_in": int(jwtExpiresIn.Seconds()),
		})
	}
}

// handleLogout 退出登录
func handleLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 实际应用中可以在此实现token黑名单逻辑
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "退出登录成功",
		})
	}
}

// handleMe 获取当前用户信息
func handleMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"user": gin.H{
				"id":       c.MustGet("user_id"),
				"username": c.MustGet("username"),
				"role":     c.MustGet("role"),
			},
		})
	}
}

// handleChangePassword 修改密码
func handleChangePassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		userID := c.MustGet("user_id").(uint)
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "USER_NOT_FOUND", "message": "用户不存在"}})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "INVALID_OLD_PASSWORD", "message": "原密码错误"}})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": "密码加密失败"}})
			return
		}

		if err := db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": "密码更新失败"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码修改成功"})
	}
}

// handleCreateServer 创建服务器 (需要认证)
func handleCreateServer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Address     string `json:"address" binding:"required"`
			Port        int    `json:"port"`
			Type        string `json:"type"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		if req.Port == 0 {
			req.Port = 25565
		}
		if req.Type == "" {
			req.Type = "auto"
		}

		server := models.Server{
			Name:        req.Name,
			Address:     req.Address,
			Port:        req.Port,
			Type:        req.Type,
			Description: req.Description,
			Status:      "checking",
		}

		if err := db.Create(&server).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "创建服务器失败"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "data": server})
	}
}

// handleUpdateServer 更新服务器 (需要认证)
func handleUpdateServer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var server models.Server

		if err := db.First(&server, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "NOT_FOUND", "message": "服务器不存在"}})
			return
		}

		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "VALIDATION_ERROR", "message": "请求参数验证失败"}})
			return
		}

		if req.Name != nil {
			server.Name = *req.Name
		}
		if req.Description != nil {
			server.Description = *req.Description
		}

		if err := db.Save(&server).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "更新服务器失败"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": server})
	}
}

// handleDeleteServer 删除服务器 (需要认证)
func handleDeleteServer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Delete(&models.Server{}, id)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "删除服务器失败"}})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "NOT_FOUND", "message": "服务器不存在"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "服务器删除成功"})
	}
}

// handlePingServer 手动ping服务器并更新状态 (需要认证)
func handlePingServer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var server models.Server

		if err := db.First(&server, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "NOT_FOUND", "message": "服务器不存在"}})
			return
		}

		var serverInfo *services.MinecraftServer
		var err error
		var detectedType string

		switch server.Type {
		case "java":
			serverInfo, err = services.JavaServerPing(server.Address, server.Port)
		case "bedrock":
			serverInfo, err = services.BedrockServerPing(server.Address, server.Port)
		case "auto":
			serverInfo, detectedType, err = services.AutoDetectServer(server.Address, server.Port, 19132)
			if err == nil && detectedType != "" {
				db.Model(&server).Update("type", detectedType)
			}
		default:
			serverInfo, err = services.JavaServerPing(server.Address, server.Port)
		}

		if err != nil {
			db.Model(&server).Update("status", "offline")
			c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": map[string]interface{}{"code": "SERVER_UNREACHABLE", "message": "无法连接到服务器", "details": err.Error()}})
			return
		}

		db.Model(&server).Updates(map[string]interface{}{"status": "online"})
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": map[string]interface{}{
				"status":         "online",
				"players_online": serverInfo.Players.Online,
				"max_players":    serverInfo.Players.Max,
				"ping":           serverInfo.Ping,
				"version":        serverInfo.Version.Name,
			},
		})
	}
}

// -------------------------
// Statistics & User Management (Admin-level)
// -------------------------

// handleGetUsers 获取用户列表 (需要Admin权限)
func handleGetUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		// 避免返回密码哈希
		if err := db.Select("id", "username", "role", "created_at", "updated_at", "last_login_at").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "获取用户列表失败"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
	}
}

// handleCreateUser 创建用户 (需要Admin权限)
func handleCreateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required,min=6"`
			Role     string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": "密码加密失败"}})
			return
		}

		role := req.Role
		if role == "" {
			role = "user" // 默认角色
		}

		user := models.User{Username: req.Username, Password: string(hashedPassword), Role: role}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "创建用户失败"}})
			return
		}

		// 避免返回密码哈希
		user.Password = ""
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": user})
	}
}

// handleUpdateUser 更新用户 (需要Admin权限)
func handleUpdateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var user models.User
		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "NOT_FOUND", "message": "用户不存在"}})
			return
		}

		var req struct {
			Username *string `json:"username"`
			Role     *string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": map[string]interface{}{"code": "VALIDATION_ERROR", "message": err.Error()}})
			return
		}

		if req.Username != nil {
			user.Username = *req.Username
		}
		if req.Role != nil {
			user.Role = *req.Role
		}

		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "更新用户失败"}})
			return
		}
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
	}
}

// handleDeleteUser 删除用户 (需要Admin权限)
func handleDeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Delete(&models.User{}, id)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": map[string]interface{}{"code": "DATABASE_ERROR", "message": "删除用户失败"}})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": map[string]interface{}{"code": "NOT_FOUND", "message": "用户不存在"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "用户删除成功"})
	}
}

// -------------------------
// Helper Functions
// -------------------------

// sampleStats 对统计数据进行采样
func sampleStats(stats []models.ServerStat, maxPoints int) []models.ServerStat {
	if len(stats) <= maxPoints {
		return stats
	}
	step := float64(len(stats)) / float64(maxPoints)
	var sampled []models.ServerStat
	for i := 0.0; i < float64(len(stats)); i += step {
		sampled = append(sampled, stats[int(i)])
	}
	return sampled
}

// calculateStatsSummary 计算统计数据的摘要信息
func calculateStatsSummary(stats []models.ServerStat) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{"avg_players": 0, "max_players": 0, "avg_ping": 0, "uptime": 0}
	}
	var totalPlayers, totalPing, onlineCount, maxPlayers int
	for _, stat := range stats {
		totalPlayers += stat.PlayersOnline
		if stat.Ping > 0 {
			totalPing += stat.Ping
			onlineCount++
		}
		if stat.PlayersOnline > maxPlayers {
			maxPlayers = stat.PlayersOnline
		}
	}
	avgPlayers := float64(totalPlayers) / float64(len(stats))
	var avgPing float64
	if onlineCount > 0 {
		avgPing = float64(totalPing) / float64(onlineCount)
	}
	uptime := float64(onlineCount) / float64(len(stats)) * 100
	return map[string]interface{}{"avg_players": avgPlayers, "max_players": maxPlayers, "avg_ping": avgPing, "uptime": uptime}
}

// getIntervalString 获取时间间隔字符串
func getIntervalString(timeRange string) string {
	switch timeRange {
	case "30m", "1h", "6h":
		return "1 MINUTE"
	case "24h":
		return "5 MINUTE"
	case "7d":
		return "1 HOUR"
	case "30d":
		return "6 HOUR"
	default:
		return "5 MINUTE"
	}
}
