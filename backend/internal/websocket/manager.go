package websocket

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Message 表示WebSocket消息
type Message struct {
	Type      string      `json:"type"`
	ServerID  *uint       `json:"server_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// Client 表示一个WebSocket客户端
type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan Message
	Hub    *Hub
	UserID *uint // 如果是认证用户
}

// Hub 管理所有WebSocket连接
type Hub struct {
	// 注册的客户端
	clients map[*Client]bool

	// 从客户端接收的消息
	broadcast chan Message

	// 注册请求
	register chan *Client

	// 注销请求
	unregister chan *Client

	// 互斥锁
	mutex sync.RWMutex
	
	// 统计信息
	stats struct {
		TotalConnections    int64
		ActiveConnections   int
		MessagesSent        int64
		MessagesDropped     int64
		mutex              sync.RWMutex
	}
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 1000), // 增加缓冲区大小
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
	}
}

// Run 启动Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			
			// 更新统计信息
			h.stats.mutex.Lock()
			h.stats.TotalConnections++
			h.stats.ActiveConnections = len(h.clients)
			h.stats.mutex.Unlock()
			
			log.Printf("WebSocket client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			activeCount := len(h.clients)
			h.mutex.Unlock()
			
			// 更新统计信息
			h.stats.mutex.Lock()
			h.stats.ActiveConnections = activeCount
			h.stats.mutex.Unlock()
			
			log.Printf("WebSocket client disconnected: %s", client.ID)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage 向所有客户端广播消息
func (h *Hub) broadcastMessage(message Message) {
	h.mutex.RLock()
	clientCount := len(h.clients)
	clients := make([]*Client, 0, clientCount)
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()
	
	if clientCount == 0 {
		return
	}
	
	var sentCount, droppedCount int64
	
	// 并发发送消息
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			select {
			case c.Send <- message:
				sentCount++
			default:
				// 客户端缓冲区满，丢弃消息
				droppedCount++
				log.Printf("Message dropped for client %s", c.ID)
			}
		}(client)
	}
	
	wg.Wait()
	
	// 更新统计信息
	h.stats.mutex.Lock()
	h.stats.MessagesSent += sentCount
	h.stats.MessagesDropped += droppedCount
	h.stats.mutex.Unlock()
}

// BroadcastMessage 广播消息给所有客户端
func (h *Hub) BroadcastMessage(message Message) {
	select {
	case h.broadcast <- message:
	default:
		// 广播队列满，记录错误
		log.Printf("Broadcast queue full, message dropped: %s", message.Type)
		h.stats.mutex.Lock()
		h.stats.MessagesDropped++
		h.stats.mutex.Unlock()
	}
}

// GetStats 获取WebSocket统计信息
func (h *Hub) GetStats() map[string]interface{} {
	h.stats.mutex.RLock()
	defer h.stats.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_connections":  h.stats.TotalConnections,
		"active_connections": h.stats.ActiveConnections,
		"messages_sent":      h.stats.MessagesSent,
		"messages_dropped":   h.stats.MessagesDropped,
	}
}

// GetClientCount 获取连接的客户端数量
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// Upgrader WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// HandleWebSocket 处理WebSocket连接
func (h *Hub) HandleWebSocket() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 升级HTTP连接到WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			return
		}

		// 创建客户端
		client := &Client{
			ID:   generateClientID(),
			Conn: conn,
			Send: make(chan Message, 256),
			Hub:  h,
		}

		// 注册客户端
		h.register <- client

		// 启动goroutines
		go client.writePump()
		go client.readPump()
	})
}

// readPump 处理从客户端读取消息
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// 设置读取限制和超时
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		var message Message
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.ID, err)
			}
			break
		}

		// 处理收到的消息
		log.Printf("Received message from client %s: %+v", c.ID, message)
	}
}

// writePump 处理向客户端写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error for client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// generateClientID 生成客户端ID
func generateClientID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("client_%x", b)
}

// 全局Hub实例
var GlobalHub *Hub

// InitWebSocket 初始化WebSocket
func InitWebSocket() {
	GlobalHub = NewHub()
	go GlobalHub.Run()
}

// BroadcastServerStatus 广播服务器状态更新
func BroadcastServerStatus(serverID uint, data interface{}) {
	if GlobalHub != nil {
		message := Message{
			Type:      "server_status",
			ServerID:  &serverID,
			Data:      data,
			Timestamp: getCurrentTimestamp(),
		}
		GlobalHub.BroadcastMessage(message)
	}
}

// BroadcastPlayerJoin 广播玩家加入
func BroadcastPlayerJoin(serverID uint, data interface{}) {
	if GlobalHub != nil {
		message := Message{
			Type:      "player_join",
			ServerID:  &serverID,
			Data:      data,
			Timestamp: getCurrentTimestamp(),
		}
		GlobalHub.BroadcastMessage(message)
	}
}

// BroadcastPlayerLeave 广播玩家离开
func BroadcastPlayerLeave(serverID uint, data interface{}) {
	if GlobalHub != nil {
		message := Message{
			Type:      "player_leave",
			ServerID:  &serverID,
			Data:      data,
			Timestamp: getCurrentTimestamp(),
		}
		GlobalHub.BroadcastMessage(message)
	}
}

// BroadcastStatsUpdate 广播统计数据更新
func BroadcastStatsUpdate(data interface{}) {
	if GlobalHub != nil {
		message := Message{
			Type:      "stats_update",
			Data:      data,
			Timestamp: getCurrentTimestamp(),
		}
		GlobalHub.BroadcastMessage(message)
	}
}

func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}