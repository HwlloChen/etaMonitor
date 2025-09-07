package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	
	// Clean up expired entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	
	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	for ip, requests := range rl.requests {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) <= rl.window {
				validRequests = append(validRequests, reqTime)
			}
		}
		
		if len(validRequests) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validRequests
		}
	}
}

// getClientIP 智能获取客户端真实IP
// 优先检查反向代理头，但只有在请求来源是可信代理时才使用
func getClientIP(c *gin.Context) string {
	// 获取远程地址
	remoteAddr := c.Request.RemoteAddr
	if remoteAddr == "" {
		return "unknown"
	}
	
	// 提取IP部分（去掉端口）
	remoteIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// 如果没有端口，直接使用原地址
		remoteIP = remoteAddr
	}
	
	// 检查是否为可信的代理IP
	if !isTrustedProxy(remoteIP) {
		// 如果不是可信代理，直接返回远程IP（直接访问场景）
		return remoteIP
	}
	
	// 如果是可信代理，尝试从头部获取真实IP
	// 优先级：X-Real-IP > X-Forwarded-For 的第一个IP
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" && isValidIP(realIP) {
		return realIP
	}
	
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个（客户端IP）
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if isValidIP(clientIP) {
				return clientIP
			}
		}
	}
	
	// 如果代理头无效，回退到远程IP
	return remoteIP
}

// isTrustedProxy 检查IP是否为可信代理
func isTrustedProxy(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	
	// 定义可信的代理网络范围
	trustedNets := []string{
		"127.0.0.0/8",    // localhost
		"10.0.0.0/8",     // 私有网络 10.x.x.x
		"172.16.0.0/12",  // 私有网络 172.16-31.x.x
		"192.168.0.0/16", // 私有网络 192.168.x.x
	}
	
	for _, netStr := range trustedNets {
		_, network, err := net.ParseCIDR(netStr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}
	
	return false
}

// isValidIP 检查IP是否有效
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func (rl *rateLimiter) isAllowed(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// Get requests for this IP
	requests, exists := rl.requests[ip]
	if !exists {
		rl.requests[ip] = []time.Time{now}
		return true
	}
	
	// Filter out expired requests
	var validRequests []time.Time
	for _, reqTime := range requests {
		if now.Sub(reqTime) <= rl.window {
			validRequests = append(validRequests, reqTime)
		}
	}
	
	// Check if limit is exceeded
	if len(validRequests) >= rl.limit {
		return false
	}
	
	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	
	return true
}

func (rl *rateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		
		if !rl.isAllowed(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "请求过于频繁，请稍后再试",
				},
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}