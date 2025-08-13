package api

import (
	"log"

	"etamonitor/internal/websocket"
	"github.com/gin-gonic/gin"
)

// handleWebSocket 处理WebSocket连接请求
func handleWebSocket(c *gin.Context) {
	if websocket.GlobalHub == nil {
		log.Printf("WebSocket Hub未初始化")
		c.Status(500)
		return
	}

	// 使用全局Hub处理WebSocket连接
	websocket.GlobalHub.HandleWebSocket()(c)
}

// getWebSocketStats 获取WebSocket连接统计
func handleWebSocketStats(c *gin.Context) {
	if websocket.GlobalHub == nil {
		c.JSON(500, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": "WebSocket服务未初始化",
			},
		})
		return
	}

	clientCount := websocket.GlobalHub.GetClientCount()
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"connected_clients": clientCount,
			"status":           "running",
		},
	})
}