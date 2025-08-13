package api

import (
	"errors"
	"net/http"
	"strconv"

	"etamonitor/internal/models"
	"etamonitor/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// =================================================================================
// Public Handlers (无需认证)
//
// 此文件包含所有公开的、无需用户认证即可访问的API处理器。
// 主要用于提供服务器和玩家的公开信息查询。
// =================================================================================

// handleGetServers 获取服务器列表
func handleGetServers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		status := c.Query("status")

		offset := (page - 1) * limit

		query := db.Model(&models.Server{})
		if status != "" {
			query = query.Where("status = ?", status)
		}

		var servers []models.Server
		var total int64

		query.Count(&total)
		query.Offset(offset).Limit(limit).Find(&servers)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    servers,
			"pagination": gin.H{
				"current_page": page,
				"total_pages":  (int(total) + limit - 1) / limit,
				"total_count":  total,
				"per_page":     limit,
			},
		})
	}
}

// handleGetServer 获取单个服务器的详细信息
func handleGetServer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// 验证ID格式
		serverID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ID",
					"message": "服务器ID格式无效",
				},
			})
			return
		}

		var server models.Server
		if err := db.First(&server, serverID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "NOT_FOUND",
						"message": "服务器不存在",
					},
				})
				return
			}
			// 其他数据库错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "DATABASE_ERROR",
					"message": "数据库查询失败",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    server,
		})
	}
}

// handleDetectServer 检测服务器类型和状态
func handleDetectServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Address string `json:"address" binding:"required"`
			Port    int    `json:"port"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "VALIDATION_ERROR",
					"message": "请求参数验证失败",
					"details": err.Error(),
				},
			})
			return
		}

		if req.Port == 0 {
			req.Port = 25565
		}

		// 执行自动检测
		serverInfo, detectedType, err := services.AutoDetectServer(
			req.Address,
			req.Port,
			19132,
		)

		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "SERVER_UNREACHABLE",
					"message": "无法检测服务器类型",
					"details": err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": map[string]interface{}{
				"type":           detectedType,
				"server_type":    serverInfo.ServerType.String(),
				"version":        serverInfo.Version.Name,
				"protocol":       serverInfo.Version.Protocol,
				"online":         true,
				"players_online": serverInfo.Players.Online,
				"max_players":    serverInfo.Players.Max,
				"ping":           serverInfo.Ping,
				"motd":           getDescriptionText(serverInfo.Description),
				"players_sample": serverInfo.Players.Sample,
				"favicon":        serverInfo.Favicon,
			},
		})
	}
}

// handleGetPlayers 获取玩家列表
func handleGetPlayers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var players []models.Player
		db.Order("total_playtime desc").Find(&players)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    players,
		})
	}
}

// handleGetPlayer 获取单个玩家的详细信息
func handleGetPlayer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		player, err := getPlayerByUUIDOrUsername(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "玩家不存在",
			})
			return
		}

		// 统计所有会话的 duration 总和
		var totalPlaytime int64
		db.Model(&models.PlayerSession{}).Where("player_id = ? AND duration > 0", player.ID).Select("SUM(duration)").Scan(&totalPlaytime)

		result := map[string]interface{}{
			"id":             player.ID,
			"username":       player.Username,
			"uuid":           player.UUID,
			"first_seen":     player.FirstSeen,
			"last_seen":      player.LastSeen,
			"total_playtime": totalPlaytime,
			"rank":           player.Rank,
			"created_at":     player.CreatedAt,
			"updated_at":     player.UpdatedAt,
			"avatar":         getPlayerAvatar(player.UUID, player.Username),
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    result,
		})
	}
}

// handleGetPlayerSessions 获取玩家的游戏会话列表
func handleGetPlayerSessions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		serverID := c.Query("server_id")

		player, err := getPlayerByUUIDOrUsername(db, id)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    []interface{}{},
			})
			return
		}

		var sessions []models.PlayerSession
		query := db.Preload("Server").Where("player_id = ?", player.ID)
		if serverID != "" {
			query = query.Where("server_id = ?", serverID)
		}
		if err := query.Order("join_time desc").Find(&sessions).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "查询玩家会话失败",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    sessions,
		})
	}
}

// handleGetServerOnlinePlayers 获取服务器当前在线玩家列表
func handleGetServerOnlinePlayers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")

		// 验证ID格式
		id, err := strconv.ParseUint(serverID, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ID",
					"message": "服务器ID格式无效",
				},
			})
			return
		}

		// 获取服务器信息
		var server models.Server
		if err := db.First(&server, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "SERVER_NOT_FOUND",
						"message": "服务器不存在",
					},
				})
				return
			}
			// 其他数据库错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "DATABASE_ERROR",
					"message": "数据库查询失败",
				},
			})
			return
		}

		// 获取当前活跃的会话（在线玩家）
		var activeSessions []models.PlayerSession
		if err := db.Preload("Player").Where("server_id = ? AND leave_time IS NULL", serverID).Find(&activeSessions).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INTERNAL_ERROR",
					"message": "获取在线玩家失败",
				},
			})
			return
		}

		// 构建在线玩家列表
		var onlinePlayers []map[string]interface{}
		for _, session := range activeSessions {
			player := map[string]interface{}{
				"username": session.Player.Username,
				"uuid":     session.Player.UUID,
				"rank":     session.Player.Rank,
				"joinTime": session.JoinTime,
				"avatar":   getPlayerAvatar(session.Player.UUID, session.Player.Username),
			}
			onlinePlayers = append(onlinePlayers, player)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    onlinePlayers,
		})
	}
}

// -------------------------
// Helper Functions
// -------------------------

// getDescriptionText 从Description结构体中提取文本
func getDescriptionText(desc services.Description) string {
	if desc.Text != "" {
		return desc.Text
	}
	// 如果Text为空，尝试从Extra中提取
	if len(desc.Extra) > 0 {
		var result string
		for _, extra := range desc.Extra {
			result += extra.Text
		}
		return result
	}
	return "Minecraft Server"
}

// getPlayerAvatar 生成玩家头像URL
func getPlayerAvatar(uuid, username string) string {
	if username != "" {
		// Fallback for offline mode servers
		return "https://crafthead.net/avatar/" + username
	}
	if uuid != "" {
		return "https://crafthead.net/avatar/" + uuid
	}
	// Default Steve avatar
	return "https://crafthead.net/avatar/steve"
}

// getPlayerByUUIDOrUsername 通过 UUID 或 Username 获取玩家信息
func getPlayerByUUIDOrUsername(db *gorm.DB, id string) (*models.Player, error) {
	var player models.Player
	// 优先使用 UUID 查询
	err := db.Where("uuid = ?", id).First(&player).Error
	if err == nil {
		return &player, nil
	}
	// 如果 UUID 查询失败，并且错误不是“记录未找到”，则返回错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// 尝试使用 username 查询
	err = db.Where("username = ?", id).First(&player).Error
	if err != nil {
		return nil, err // 返回最终的错误（可能是 gorm.ErrRecordNotFound）
	}
	return &player, nil
}
