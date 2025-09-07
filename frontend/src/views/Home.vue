<template>
  <div class="home">
    <div class="stats-grid">
      <!-- 统计卡片 -->
      <mdui-card class="stat-card" v-for="(stat, index) in statCards" :key="stat.key">
        <div class="stat-content">
          <mdui-icon :name="stat.icon" :class="['stat-icon', stat.iconClass]"></mdui-icon>
          <div class="stat-info">
            <div class="stat-number">
              <span v-if="statsLoaded">{{ stats[stat.key] }}</span>
              <mdui-circular-progress v-else></mdui-circular-progress>
            </div>
            <div class="stat-label">{{ stat.label }}</div>
          </div>
        </div>
      </mdui-card>
    </div>

    <!-- 列表容器 -->
    <div class="lists-container">
      <!-- 服务器列表 -->
      <mdui-card class="server-list-card">
        <div class="card-header">
          <h2>服务器列表</h2>
          <mdui-button variant="outlined" icon="refresh" @click="refreshServers" :loading="serversRefreshing">
            刷新
          </mdui-button>
        </div>

        <!-- 加载中状态 -->
        <div v-if="serversLoading" class="loading-state">
          <mdui-circular-progress></mdui-circular-progress>
          <p>加载服务器列表中...</p>
        </div>

        <!-- 服务器列表 -->
        <mdui-list v-else-if="servers.length > 0">
          <mdui-list-item v-for="server in servers" :key="server.id" @click="goToServer(server.id)" class="server-item">
            <mdui-avatar slot="icon">
              <mdui-icon :name="server.status === 'online' ? 'wifi_tethering' : 'wifi_off'"></mdui-icon>
            </mdui-avatar>

            <div class="server-info">
              <div class="server-name">{{ server.name }}</div>
              <div class="server-address">{{ server.address }}:{{ server.port }}</div>
            </div>

            <div class="server-status" slot="end-icon">
              <mdui-chip>
                {{ server.status === 'online' ? '在线' : '离线' }}
              </mdui-chip>
              <div v-if="server.status === 'online'" class="player-count">
                {{ server.players_online || 0 }}/{{ server.max_players || 0 }} 玩家
              </div>
            </div>
          </mdui-list-item>
        </mdui-list>

        <!-- 空状态 -->
        <div v-else class="empty-state">
          <mdui-icon name="dns_off" class="empty-icon"></mdui-icon>
          <p>暂无服务器</p>
        </div>
      </mdui-card>

      <!-- 最近活跃玩家 -->
      <mdui-card class="recent-players-card">
        <div class="card-header">
          <h2>最近活跃玩家</h2>
        </div>

        <!-- 加载中状态 -->
        <div v-if="playersLoading" class="loading-state">
          <mdui-circular-progress></mdui-circular-progress>
          <p>加载玩家活动中...</p>
        </div>

        <!-- 玩家活动列表 -->
        <mdui-list v-else-if="playerStore.recentActivities.length > 0">
          <mdui-list-item v-for="(activity, index) in playerStore.getRecentActivities(8)"
            :key="`${activity.playerId}-${activity.timestamp}-${index}`" @click="goToPlayer(activity.playerName)">
            <mdui-avatar slot="icon" :src="activity.playerAvatar"></mdui-avatar>

            <div class="player-info">
              <div class="player-name">{{ activity.playerName }}</div>
              <div class="player-action">
                <mdui-icon :name="activity.type === 'join' ? 'login' : 'logout'"
                  :class="activity.type === 'join' ? 'join-icon' : 'leave-icon'"></mdui-icon>
                {{ activity.type === 'join' ? '加入' : '离开' }} {{ activity.serverName }}
                <span v-if="activity.type === 'leave' && activity.sessionDuration" class="duration">
                  ({{ playerStore.formatDuration(activity.sessionDuration) }})
                </span>
              </div>
            </div>

            <div class="activity-time" slot="end-icon">
              {{ formatTime(activity.timestamp) }}
            </div>
          </mdui-list-item>
        </mdui-list>

        <!-- 空状态 -->
        <div v-else class="empty-state">
          <mdui-icon name="person_off" class="empty-icon"></mdui-icon>
          <p>暂无玩家活动</p>
        </div>
      </mdui-card>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useServerStore } from '../stores/server'
import { usePlayerStore } from '../stores/player'
import { getWebSocketStatus, getWebSocketStatusText } from '../utils/ws'

export default {
  name: 'Home',
  setup() {
    const router = useRouter()
    const serverStore = useServerStore()
    const playerStore = usePlayerStore()

    const stats = ref({
      totalServers: 0,
      onlineServers: 0,
      totalPlayers: 0,
      peakPlayers: 0
    })

    const servers = ref([])
    const wsStatus = ref('disconnected')
    const wsStatusText = ref('连接断开')

    // 加载状态
    const statsLoaded = ref(false)
    const serversLoading = ref(false)
    const playersLoading = ref(false)
    const serversRefreshing = ref(false)

    // 统计卡片配置
    const statCards = [
      {
        key: 'totalServers',
        icon: 'dns',
        label: '总服务器数',
        iconClass: ''
      },
      {
        key: 'onlineServers',
        icon: 'wifi_tethering',
        label: '在线服务器',
        iconClass: 'online'
      },
      {
        key: 'totalPlayers',
        icon: 'people',
        label: '总玩家数',
        iconClass: ''
      },
      {
        key: 'peakPlayers',
        icon: 'trending_up',
        label: '今日峰值',
        iconClass: ''
      }
    ]

    // 事件监听器清理函数
    const eventCleanupFunctions = []

    const loadStats = async () => {
      try {
        const response = await serverStore.getStats()
        if (response && response.data) {
          stats.value = response.data
        }
      } catch (error) {
        console.error('加载统计数据失败:', error)
      } finally {
        statsLoaded.value = true // 后续加载状态不再显示
      }
    }

    const loadServers = async (showLoading = true) => {
      if (showLoading) {
        serversLoading.value = true
      }

      try {
        const response = await serverStore.getServers()
        if (response && response.data) {
          servers.value = response.data
        }
      } catch (error) {
        console.error('加载服务器列表失败:', error)
      } finally {
        if (showLoading) {
          serversLoading.value = false
        }
      }
    }

    const loadPlayers = async () => {
      playersLoading.value = true
      try {
        // 从API加载最近活动数据
        await playerStore.fetchRecentActivities({ limit: 20 })
      } catch (error) {
        console.error('加载玩家活动失败:', error)
      } finally {
        playersLoading.value = false
      }
    }

    const refreshServers = async () => {
      serversRefreshing.value = true
      try {
        await Promise.all([
          loadServers(false),
          loadStats(),
          playerStore.fetchRecentActivities({ limit: 20 })
        ])
      } catch (error) {
        console.error('刷新数据失败:', error)
      } finally {
        serversRefreshing.value = false
      }
    }

    const goToServer = (id) => {
      router.push(`/server/${id}`)
    }

    const goToPlayer = (playerId) => {
      router.push(`/player/${playerId}`)
    }

    const formatTime = (timestamp) => {
      const date = new Date(timestamp)
      const now = new Date()
      const diff = now - date

      if (diff < 60000) { // 1分钟内
        return '刚刚'
      } else if (diff < 3600000) { // 1小时内
        return `${Math.floor(diff / 60000)}分钟前`
      } else if (diff < 86400000) { // 1天内
        return `${Math.floor(diff / 3600000)}小时前`
      } else {
        return date.toLocaleString('zh-CN', {
          month: 'numeric',
          day: 'numeric',
          hour: 'numeric',
          minute: 'numeric'
        })
      }
    }

    const updateWebSocketStatus = () => {
      wsStatus.value = getWebSocketStatus()
      wsStatusText.value = getWebSocketStatusText()
    }

    // 设置事件监听器
    const setupEventListeners = () => {
      // 监听服务器状态更新
      const onServerStatusUpdate = (event) => {
        const { serverId, data } = event.detail

        // 更新本地服务器列表
        const serverIndex = servers.value.findIndex(s => s.id === serverId)
        if (serverIndex !== -1) {
          servers.value[serverIndex] = {
            ...servers.value[serverIndex],
            ...data
          }
        }

        // 重新加载统计数据（不显示加载状态）
        loadStats()
      }

      // 监听玩家加入/离开事件
      const onPlayerActivity = () => {
        // 重新加载统计数据和服务器列表以更新在线人数（不显示加载状态）
        loadStats()
        loadServers(false)
      }

      // 监听统计数据更新
      const onStatsUpdate = (event) => {
        const { data } = event.detail
        if (data) {
          stats.value = { ...stats.value, ...data }
        }
      }

      // WebSocket连接状态变化监听
      const onWebSocketStatusChange = () => {
        updateWebSocketStatus()
      }

      // 添加事件监听器
      window.addEventListener('server-status-update', onServerStatusUpdate)
      window.addEventListener('player-join', onPlayerActivity)
      window.addEventListener('player-leave', onPlayerActivity)
      window.addEventListener('stats-update', onStatsUpdate)

      // 监听WebSocket状态变化（通过定时器检查）
      const statusCheckTimer = setInterval(onWebSocketStatusChange, 1000)

      // 保存清理函数
      eventCleanupFunctions.push(
        () => window.removeEventListener('server-status-update', onServerStatusUpdate),
        () => window.removeEventListener('player-join', onPlayerActivity),
        () => window.removeEventListener('player-leave', onPlayerActivity),
        () => window.removeEventListener('stats-update', onStatsUpdate),
        () => clearInterval(statusCheckTimer)
      )
    }

    const cleanupEventListeners = () => {
      eventCleanupFunctions.forEach(cleanup => cleanup())
      eventCleanupFunctions.length = 0
    }

    onMounted(async () => {
      // 并行加载初始数据
      await Promise.all([
        loadStats(),
        loadServers(),
        loadPlayers()
      ])

      // 设置事件监听器
      setupEventListeners()

      // 初始化WebSocket状态
      updateWebSocketStatus()
    })

    onUnmounted(() => {
      // 清理事件监听器
      cleanupEventListeners()
    })

    return {
      stats,
      servers,
      serverStore,
      playerStore,
      wsStatus,
      wsStatusText,
      statsLoaded,
      serversLoading,
      playersLoading,
      serversRefreshing,
      statCards,
      refreshServers,
      goToServer,
      goToPlayer,
      formatTime
    }
  }
}
</script>

<style scoped>
.home {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.lists-container {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
  margin-bottom: 24px;
}

@media (min-width: 1024px) {
  .lists-container {
    grid-template-columns: 3fr 2fr;
  }
}

.stat-card {
  padding: 24px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: clamp(12px, 2vw, 16px);
}

.stat-icon {
  font-size: clamp(36px, 4vw, 48px);
  color: rgb(var(--mdui-color-primary));
}

.stat-icon.online {
  color: rgb(var(--mdui-color-success));
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-number {
  font-size: clamp(24px, 3vw, 32px);
  font-weight: bold;
  color: rgb(var(--mdui-color-on-surface));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  min-height: 1.2em;
}

.stat-label {
  font-size: clamp(12px, 1.5vw, 14px);
  color: rgb(var(--mdui-color-on-surface-variant));
  margin-top: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  gap: 16px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

:deep(mdui-avatar) {
  border-radius: var(--mdui-shape-corner-extra-small);
  overflow: hidden;
  width: 2.4rem;
  height: 2.4rem;
}

:deep(mdui-avatar img) {
  image-rendering: pixelated;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px 24px 16px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
}

.server-item {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.server-item:hover {
  background-color: rgb(var(--mdui-color-surface-variant));
}

.server-info {
  flex: 1;
}

.server-name {
  font-size: 16px;
  font-weight: 500;
  margin-bottom: 4px;
}

.server-address {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.server-status {
  text-align: right;
}

.player-count {
  font-size: 12px;
  color: rgb(var(--mdui-color-on-surface-variant));
  margin-top: 4px;
}

.player-info {
  flex: 1;
}

.player-name {
  font-size: 16px;
  font-weight: 500;
}

.player-server {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.player-time {
  font-size: 12px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.activity-time {
  font-size: 12px;
  color: rgb(var(--mdui-color-on-surface-variant));
  text-align: right;
}

.player-action {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
  display: flex;
  align-items: center;
  gap: 4px;
}

.join-icon {
  color: rgb(var(--mdui-color-success));
}

.leave-icon {
  color: rgb(var(--mdui-color-error));
}

.duration {
  color: rgb(var(--mdui-color-on-surface-variant));
  font-size: 12px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.empty-icon {
  font-size: 48px;
  opacity: 0.5;
  margin-bottom: 16px;
}

.websocket-status {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 1000;
}

mdui-list {
  max-height: 28rem;
  overflow-y: auto;
}

@media (max-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .home {
    padding: 16px;
  }

  .lists-container {
    gap: 16px;
  }

  .websocket-status {
    bottom: 80px;
  }

  .player-action {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .server-info,
  .player-info {
    min-width: 0;
  }

  .server-name,
  .player-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  mdui-list {
    max-height: 20rem;
  }
}
</style>