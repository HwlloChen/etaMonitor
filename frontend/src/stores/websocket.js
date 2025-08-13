import { defineStore } from 'pinia'

export const useWebSocketStore = defineStore('websocket', {
  state: () => ({
    socket: null,
    isConnected: false,
    reconnectTimer: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    reconnectInterval: 3000,
    listeners: new Map(), // 消息类型监听器
  }),

  actions: {
    // 连接WebSocket
    connect() {
      if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        return
      }

      try {
        // 构建WebSocket URL
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/ws`
        
        this.socket = new WebSocket(wsUrl)

        // 连接成功
        this.socket.addEventListener('open', () => {
          this.isConnected = true
          this.reconnectAttempts = 0
          
          // 清除重连定时器
          if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
            this.reconnectTimer = null
          }
        })

        // 接收消息
        this.socket.addEventListener('message', (event) => {
          try {
            const message = JSON.parse(event.data)
            this.handleMessage(message)
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error, event.data)
          }
        })

        // 连接关闭
        this.socket.addEventListener('close', (event) => {
          this.isConnected = false
          this.socket = null
          
          // 自动重连
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect()
          }
        })

        // 连接错误
        this.socket.addEventListener('error', (error) => {
          console.error('WebSocket error:', error)
          this.isConnected = false
        })

      } catch (error) {
        console.error('Failed to create WebSocket connection:', error)
      }
    },

    // 断开连接
    disconnect() {
      
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
      }

      if (this.socket) {
        this.socket.close()
        this.socket = null
      }
      
      this.isConnected = false
      this.reconnectAttempts = 0
    },

    // 发送消息
    send(message) {
      if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        try {
          this.socket.send(JSON.stringify(message))
          return true
        } catch (error) {
          console.error('Failed to send WebSocket message:', error)
          return false
        }
      } else {
        console.warn('WebSocket not connected, cannot send message')
        return false
      }
    },

    // 处理接收到的消息
    handleMessage(message) {
      const { type, server_id, data, timestamp } = message
      
      // 调用特定类型的监听器
      if (this.listeners.has(type)) {
        const handlers = this.listeners.get(type)
        handlers.forEach(handler => {
          try {
            handler(data, server_id, timestamp)
          } catch (error) {
            console.error(`Error in ${type} listener:`, error)
          }
        })
      }

      // 调用通用监听器
      if (this.listeners.has('*')) {
        const handlers = this.listeners.get('*')
        handlers.forEach(handler => {
          try {
            handler(message)
          } catch (error) {
            console.error('Error in universal listener:', error)
          }
        })
      }
    },

    // 添加消息监听器
    addListener(type, handler) {
      if (!this.listeners.has(type)) {
        this.listeners.set(type, new Set())
      }
      this.listeners.get(type).add(handler)
      
      return () => this.removeListener(type, handler)
    },

    // 移除消息监听器
    removeListener(type, handler) {
      if (this.listeners.has(type)) {
        this.listeners.get(type).delete(handler)
        if (this.listeners.get(type).size === 0) {
          this.listeners.delete(type)
        }
      }
    },

    // 安排重连
    scheduleReconnect() {
      if (this.reconnectTimer) {
        return
      }

      this.reconnectAttempts++
      const delay = Math.min(this.reconnectInterval * this.reconnectAttempts, 30000)
      
      
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null
        this.connect()
      }, delay)
    },

    // 重置重连计数
    resetReconnectAttempts() {
      this.reconnectAttempts = 0
    }
  },

  getters: {
    connectionStatus() {
      if (!this.socket) return 'disconnected'
      
      switch (this.socket.readyState) {
        case WebSocket.CONNECTING:
          return 'connecting'
        case WebSocket.OPEN:
          return 'connected'
        case WebSocket.CLOSING:
          return 'closing'
        case WebSocket.CLOSED:
          return 'closed'
        default:
          return 'unknown'
      }
    }
  }
})