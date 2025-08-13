import { useWebSocketStore } from '../stores/websocket'
import { useServerStore } from '../stores/server'
import { usePlayerStore } from '../stores/player'
import { snackbar } from 'mdui'

class WebSocketManager {
  constructor() {
    this.wsStore = null
    this.serverStore = null
    this.playerStore = null
    this.isInitialized = false
    this.notificationPermission = 'default'
  }

  // 初始化WebSocket管理器
  async init() {
    if (this.isInitialized) return

    // 获取stores
    this.wsStore = useWebSocketStore()
    this.serverStore = useServerStore()
    this.playerStore = usePlayerStore()

    // 请求通知权限
    await this.requestNotificationPermission()

    // 连接WebSocket
    this.connect()

    // 设置事件监听器
    this.setupEventListeners()

    this.isInitialized = true
  }

  // 请求浏览器通知权限
  async requestNotificationPermission() {
    if ('Notification' in window && localStorage.getItem('notificationStatus') === null) {
      snackbar({
        message: '我们将请求通知权限，以便在玩家加入或离开时通知您。',
        autoCloseDelay: 0,
        closeable: true,
        action: '允许',
        onActionClick: () => {
          return new Promise(async (resolve, reject) => {
            const permission = await Notification.requestPermission()
            this.notificationPermission = permission
            console.log('Notification permission:', permission)
            resolve(permission)
          }).then((permission) => {
            if (permission === 'granted') {
              localStorage.setItem('notificationStatus', 'granted')
            } else {
              localStorage.setItem('notificationStatus', 'refused')
              snackbar({
                message: '请允许通知权限申请, 或下次点击右侧关闭按钮不再申请',
                closeOnOutsideClick: true,
                autoCloseDelay: 3000
              })
            }
          }).catch((error) => {
            console.error('请求通知权限失败:', error)
            localStorage.setItem('notificationStatus', 'error')
          })
        },
        onClosed: () => {
          if (localStorage.getItem('notificationStatus')) {
            if (localStorage.getItem('notificationStatus') !== 'granted') {
              localStorage.removeItem('notificationStatus') // 没有成功申请，下次再请求
              return
            }
          } else {
            // 用户关闭了 snackbar，视为拒绝
            localStorage.setItem('notificationStatus', 'refused')
          }
        }
      })
    }
    if (localStorage.getItem('notificationStatus') === 'granted') {
      const permission = await Notification.requestPermission()
      this.notificationPermission = permission
    }
  }

  // 连接WebSocket
  connect() {
    if (this.wsStore) {
      this.wsStore.connect()
    }
  }

  // 断开WebSocket
  disconnect() {
    if (this.wsStore) {
      this.wsStore.disconnect()
    }
  }

  // 设置事件监听器
  setupEventListeners() {
    if (!this.wsStore) return

    // 监听服务器状态更新
    this.wsStore.addListener('server_status', (data, serverId, timestamp) => {

      // 更新服务器store
      if (this.serverStore) {
        this.serverStore.updateServerRealtime(serverId, data)
      }

      // 触发自定义事件，让页面组件知道状态已更新
      window.dispatchEvent(new CustomEvent('server-status-update', {
        detail: { serverId, data, timestamp }
      }))
    })

    // 监听玩家加入事件
    this.wsStore.addListener('player_join', (data, serverId, timestamp) => {

      // 更新玩家store
      if (this.playerStore) {
        this.playerStore.handlePlayerJoin(data, serverId, timestamp)
      }

      snackbar({
        message: `<a href='/player/${data.username}'>${data.username}</a> 加入了 <a href='/server/${serverId}'>${data.server_name}</a>`,
        closeOnOutsideClick: true,
        autoCloseDelay: 3000,
        placement: 'top-end'
      })

      // 发送系统通知
      this.showNotification(
        '玩家加入',
        `${data.username} 加入了 ${data.server_name}`,
        'join'
      )

      // 触发自定义事件
      window.dispatchEvent(new CustomEvent('player-join', {
        detail: { serverId, data, timestamp }
      }))
    })

    // 监听玩家离开事件
    this.wsStore.addListener('player_leave', (data, serverId, timestamp) => {

      // 更新玩家store
      if (this.playerStore) {
        this.playerStore.handlePlayerLeave(data, serverId, timestamp)
      }

      snackbar({
        message: `<a href='/player/${data.username}'>${data.username}</a> 离开了 <a href='/server/${serverId}'>${data.server_name}</a>`,
        closeOnOutsideClick: true,
        autoCloseDelay: 3000,
        placement: 'top-end'
      })

      // 发送系统通知
      const duration = this.playerStore.formatDuration(data.session_duration)
      this.showNotification(
        '玩家离开',
        `${data.username} 离开了 ${data.server_name} (游戏时长: ${duration})`,
        'leave'
      )

      // 触发自定义事件
      window.dispatchEvent(new CustomEvent('player-leave', {
        detail: { serverId, data, timestamp }
      }))
    })

    // 监听统计数据更新
    this.wsStore.addListener('stats_update', (data, serverId, timestamp) => {

      // 触发自定义事件
      window.dispatchEvent(new CustomEvent('stats-update', {
        detail: { data, timestamp }
      }))
    })
  }

  // 显示系统通知
  showNotification(title, body, type = 'info') {
    // 检查通知权限
    if (this.notificationPermission !== 'granted') {
      return
    }

    try {
      const notification = new Notification(title, {
        body: body,
        icon: this.getNotificationIcon(type),
        badge: '/favicon.ico',
        tag: `etamonitor-${type}`,
        requireInteraction: false,
        silent: false
      })

      // 点击通知时聚焦窗口
      notification.onclick = () => {
        window.focus()
        notification.close()
      }

      // 10秒后自动关闭
      setTimeout(() => {
        notification.close()
      }, 10000)

    } catch (error) {
      console.error('显示通知失败:', error)
    }
  }

  // 获取通知图标
  getNotificationIcon(type) {
    switch (type) {
      case 'join':
        return '/icons/player-join.png'
      case 'leave':
        return '/icons/player-leave.png'
      case 'server':
        return '/icons/server.png'
      default:
        return '/favicon.ico'
    }
  }

  // 获取连接状态
  getConnectionStatus() {
    return this.wsStore ? this.wsStore.connectionStatus : 'disconnected'
  }

  // 获取连接状态文本
  getConnectionStatusText() {
    const status = this.getConnectionStatus()
    switch (status) {
      case 'connecting':
        return '连接中...'
      case 'connected':
        return '已连接'
      case 'closing':
        return '断开中...'
      case 'closed':
      case 'disconnected':
        return '连接断开'
      default:
        return '未知状态'
    }
  }

  // 发送消息
  send(message) {
    if (this.wsStore) {
      return this.wsStore.send(message)
    }
    return false
  }

  // 添加消息监听器
  addListener(type, handler) {
    if (this.wsStore) {
      return this.wsStore.addListener(type, handler)
    }
    return null
  }

  // 移除消息监听器
  removeListener(type, handler) {
    if (this.wsStore) {
      this.wsStore.removeListener(type, handler)
    }
  }
}

// 创建全局实例
const wsManager = new WebSocketManager()

// 导出管理器
export default wsManager

// 导出初始化函数
export const initWebSocket = () => {
  return wsManager.init()
}

// 导出常用方法
export const connectWebSocket = () => wsManager.connect()
export const disconnectWebSocket = () => wsManager.disconnect()
export const getWebSocketStatus = () => wsManager.getConnectionStatus()
export const getWebSocketStatusText = () => wsManager.getConnectionStatusText()