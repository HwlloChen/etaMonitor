<template>
  <div class="server-detail">
    <div class="server-header">
      <mdui-button variant="text" icon="arrow_back" @click="goBack">
        返回
      </mdui-button>

      <div class="server-title">
        <h1>{{ server.name }}</h1>
        <div class="server-address">{{ server.address }}:{{ server.port }}</div>
      </div>

      <div style="margin-left: auto">
        <mdui-chip :icon="server.status === 'online' ? 'wifi' : 'wifi_off'">
          {{ server.status === 'online' ? '在线' : '离线' }}
        </mdui-chip>
      </div>
    </div>

    <div class="content-grid">
      <!-- 服务器信息卡片 -->
      <mdui-card class="info-card">
        <div class="card-header">
          <h2>服务器信息</h2>
        </div>

        <div class="info-list">
          <div class="info-item">
            <span class="label">版本:</span>
            <span class="value">{{ server.version || 'Unknown' }}</span>
          </div>
          <div class="info-item">
            <span class="label">类型:</span>
            <span class="value">{{ server.type === 'java' ? 'Java版' : '基岩版' }}</span>
          </div>
          <div class="info-item">
            <span class="label">延迟:</span>
            <span class="value" :class="getPingClass(server.ping)">{{ server.ping || 0 }}ms</span>
          </div>
          <div class="info-item">
            <span class="label">在线人数:</span>
            <span class="value">
              {{ server.players_online || 0 }} / {{ server.max_players || 0 }}
            </span>
          </div>
          <div class="info-item" v-if="server.motd">
            <span class="label">描述:</span>
            <span class="value motd">{{ server.motd }}</span>
          </div>
        </div>
      </mdui-card>

      <!-- 在线玩家列表 -->
      <mdui-card class="players-card">
        <div class="card-header">
          <h2>在线玩家</h2>
        </div>

        <mdui-list v-if="onlinePlayers.length > 0" class="list">
          <mdui-list-item v-for="player in onlinePlayers" :key="player.username" @click="goToPlayer(player.username)">
            <mdui-avatar slot="icon" :src="player.avatar"></mdui-avatar>

            <div class="player-info">
              <div class="player-name">{{ player.username }}</div>
              <div class="player-rank">{{ player.rank }}</div>
            </div>

            <div class="player-time" slot="end-icon">
              {{ formatJoinTime(player.joinTime) }}
            </div>
          </mdui-list-item>
        </mdui-list>

        <div v-else class="empty-state">
          <mdui-icon name="person_off" class="empty-icon"></mdui-icon>
          <p>当前无玩家在线</p>
        </div>
      </mdui-card>

      <!-- 服务器活动记录 -->
      <mdui-card class="activity-card">
        <div class="card-header">
          <h2>最近活动</h2>
        </div>

        <mdui-list v-if="serverActivities.length > 0" class="list">
          <mdui-list-item v-for="activity in serverActivities" :key="`${activity.playerId}-${activity.timestamp}`">
            <mdui-avatar slot="icon" :src="activity.playerAvatar"></mdui-avatar>

            <div class="activity-info">
              <div class="activity-text">
                <mdui-icon :name="activity.type === 'join' ? 'login' : 'logout'"
                  :class="activity.type === 'join' ? 'join-icon' : 'leave-icon'"></mdui-icon>
                {{ activity.playerName }} {{ activity.type === 'join' ? '加入了' : '离开了' }}服务器
                <span v-if="activity.type === 'leave' && activity.sessionDuration" class="duration">
                  (游戏时长: {{ formatDuration(activity.sessionDuration) }})
                </span>
              </div>
            </div>

            <div class="activity-time" slot="end-icon">
              {{ formatTime(activity.timestamp) }}
            </div>
          </mdui-list-item>
        </mdui-list>

        <div v-else class="empty-state">
          <mdui-icon name="history" class="empty-icon"></mdui-icon>
          <p>暂无活动记录</p>
        </div>
      </mdui-card>
    </div>

    <!-- 服务器历史图表 -->
    <mdui-card class="chart-card">
      <div class="card-header">
        <h2>服务器状态历史</h2>
        <div class="chart-controls">
          <mdui-segmented-button-group ref="segmentedButtonGroup">
            <mdui-segmented-button v-for="range in timeRanges" :key="range.value" :value="range.value"
              :selected="selectedTimeRange === range.value" @click="selectTimeRange(range.value)">
              {{ range.label }}
            </mdui-segmented-button>
          </mdui-segmented-button-group>
        </div>
      </div>

      <div class="chart-container">
        <PlayerCountChart ref="chartComponent" :chart-data="chartData" :chart-options="chartOptions" :key="chartKey" />
      </div>
    </mdui-card>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useServerStore } from '../stores/server'
import { usePlayerStore } from '../stores/player'
import PlayerCountChart from '../components/PlayerCountChart.vue'

export default {
  name: 'ServerDetail',
  components: {
    PlayerCountChart
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const serverStore = useServerStore()
    const playerStore = usePlayerStore()

    const serverId = ref(parseInt(route.params.id))
    const server = ref({})
    const lastUpdate = ref(null)

    // 图表相关
    const chartComponent = ref(null)
    const segmentedButtonGroup = ref(null)
    const selectedTimeRange = ref('realtime')
    const isRealTimeMode = ref(true) // 是否为实时模式
    const chartKey = ref(0) // 用于强制重新渲染图表组件
    // 获取CSS变量的颜色值
    function getCssVar(name) {
      return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
    }
    const borderColor = `rgb(${getCssVar('--mdui-color-primary')})` // 图表边框颜色
    const backgroundColor = `rgba(${getCssVar('--mdui-color-primary-container')}, 0.5)` // 图表背景颜色
    const timeRanges = ref([
      { value: 'realtime', label: '实时' },
      { value: '1h', label: '1小时' },
      { value: '6h', label: '6小时' },
      { value: '24h', label: '24小时' },
      { value: '7d', label: '7天' },
      { value: '30d', label: '30天' }
    ])

    // 图表数据
    const chartData = ref({
      labels: [],
      datasets: [{
        label: '在线玩家数',
        data: [],
        borderColor: borderColor,
        backgroundColor: backgroundColor,
        fill: true,
        tension: 0.3,
        pointRadius: 2,
        pointHoverRadius: 4
      }]
    })

    // 图表配置
    const chartOptions = ref({
      responsive: true,
      maintainAspectRatio: false,
      animation: true, // 启用动画
      scales: {
        x: {
          display: true,
          title: {
            display: true,
            text: '时间'
          },
          ticks: {
            maxTicksLimit: 10
          }
        },
        y: {
          display: true,
          title: {
            display: true,
            text: '在线玩家数'
          },
          beginAtZero: true,
          ticks: {
            stepSize: 1,
            precision: 0
          },
        }
      },
      plugins: {
        legend: {
          position: 'top'
        },
        tooltip: {
          callbacks: {
            title: function (context) {
              return context[0].label
            },
            label: function (context) {
              return `在线玩家: ${context.parsed.y} 人`
            }
          }
        }
      }
    })

    // 事件监听器清理函数
    const eventCleanupFunctions = []

    // 计算属性
    const onlinePlayers = computed(() => {
      return playerStore.getOnlinePlayers(serverId.value) || []
    })

    const serverActivities = computed(() => {
      return playerStore.recentActivities
        .filter(activity => activity.serverId === serverId.value)
        .slice(0, 10)
    })

    const loadServer = async () => {
      try {
        const response = await serverStore.getServer(serverId.value)
        if (response && response.data) {
          server.value = response.data
        }
      } catch (error) {
        console.error('加载服务器详情失败:', error)
        // 检查是否为404错误
        if (error.response && error.response.status === 404) {
          router.push('/404')
          return
        }
        // 或者检查错误消息
        if (error.message && error.message.includes('404')) {
          router.push('/404')
          return
        }
      }
    }

    const loadOnlinePlayers = async () => {
      try {
        await playerStore.fetchOnlinePlayers(serverId.value)
      } catch (error) {
        console.error('加载在线玩家失败:', error)
        // 检查是否为404错误
        if (error.response && error.response.status === 404) {
          router.push('/404')
          return
        }
      }
    }

    const loadServerStats = async () => {
      try {
        const response = await fetch(`/api/stats/servers/${serverId.value}?range=${selectedTimeRange.value}`)
        
        // 检查响应状态
        if (response.status === 404) {
          router.push('/404')
          return
        }
        
        const result = await response.json()

        if (result.success && result.data && result.data.stats) {
          updateChartData(result.data.stats)
        } else {
          console.warn('No stats data received:', result)
        }
      } catch (error) {
        console.error('加载服务器统计失败:', error)
      }
    }

    // 为实时模式加载最近30分钟的数据作为初始数据
    const loadRecentDataForRealtime = async () => {
      try {
        const response = await fetch(`/api/stats/servers/${serverId.value}?range=30m`)
        
        // 检查响应状态
        if (response.status === 404) {
          router.push('/404')
          return
        }
        
        const result = await response.json()

        if (result.success && result.data && result.data.stats) {
          // 为实时模式设置初始数据
          const stats = result.data.stats
          const labels = []
          const data = []

          for (const stat of stats) {
            const date = new Date(stat.timestamp)
            const timeLabel = date.toLocaleTimeString('zh-CN', {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit'
            })

            labels.push(timeLabel)
            data.push(stat.players_online || 0)
          }

          // 更新图表数据
          chartData.value = {
            labels: labels,
            datasets: [{
              label: '在线玩家数',
              data: data,
              borderColor: borderColor,
              backgroundColor: backgroundColor,
              fill: true,
              tension: 0.3,
              pointRadius: 2,
              pointHoverRadius: 4
            }]
          }

        }
      } catch (error) {
        console.error('加载实时模式初始数据失败:', error)
      }
    }

    const updateChartData = (stats) => {
      const labels = []
      const data = []

      for (const stat of stats) {
        const date = new Date(stat.timestamp)
        const timeLabel = date.toLocaleTimeString('zh-CN', {
          month: 'numeric',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit'
        })

        labels.push(timeLabel)
        data.push(stat.players_online || 0)
      }

      // 更新图表数据
      chartData.value = {
        labels: labels,
        datasets: [{
          label: '在线玩家数',
          data: data,
          borderColor: borderColor,
          backgroundColor: backgroundColor,
          fill: true,
          tension: 0.3,
          pointRadius: 2,
          pointHoverRadius: 4
        }]
      }
    }

    const selectTimeRange = async (range) => {
      selectedTimeRange.value = range

      if (range === 'realtime') {
        // 切换到实时模式
        isRealTimeMode.value = true
        // 加载最近30分钟的数据作为初始数据
        await loadRecentDataForRealtime()
      } else {
        // 切换到历史数据模式
        isRealTimeMode.value = false
        // 加载指定时间范围的历史数据
        loadServerStats()
      }
    }

    // 添加实时数据点到图表
    const addDataPointToChart = (playersOnline, timestamp) => {
      try {
        // 只在实时模式下才添加新数据点
        if (!isRealTimeMode.value) {
          return
        }

        const timeLabel = new Date(timestamp).toLocaleTimeString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        })

        // 创建新的数据数组
        const newLabels = [...chartData.value.labels, timeLabel]
        const newData = [...chartData.value.datasets[0].data, playersOnline]

        // 保持数据点数量在合理范围内（最多50个点）
        if (newLabels.length > 50) {
          newLabels.shift()
          newData.shift()
        }

        // 更新图表数据
        chartData.value = {
          labels: newLabels,
          datasets: [{
            label: '在线玩家数',
            data: newData,
            borderColor: borderColor,
            backgroundColor: backgroundColor,
            fill: true,
            tension: 0.3,
            pointRadius: 2,
            pointHoverRadius: 4
          }]
        }
      } catch (error) {
        console.warn('图表实时更新失败:', error)
      }
    }

    const goBack = () => {
      router.go(-1)
    }

    const goToPlayer = (playerUsername) => {
      router.push(`/player/${playerUsername}`)
    }

    const formatTime = (timestamp) => {
      const date = new Date(timestamp)
      const now = new Date()
      const diff = now - date

      if (diff < 60000) {
        return '刚刚'
      } else if (diff < 3600000) {
        return `${Math.floor(diff / 60000)}分钟前`
      } else if (diff < 86400000) {
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

    const formatJoinTime = (joinTime) => {
      const date = new Date(joinTime)
      const now = new Date()
      const diff = now - date

      if (diff < 60000) {
        return '刚加入'
      } else if (diff < 3600000) {
        return `${Math.floor(diff / 60000)}分钟前加入`
      } else {
        return `${Math.floor(diff / 3600000)}小时前加入`
      }
    }

    const formatUpdateTime = (updateTime) => {
      return `更新于 ${formatTime(updateTime)}`
    }

    const formatDuration = (seconds) => {
      return playerStore.formatDuration(seconds)
    }

    const getPingClass = (ping) => {
      if (ping <= 0) return 'ping-unknown'
      if (ping < 50) return 'ping-good'
      if (ping < 100) return 'ping-fair'
      return 'ping-poor'
    }

    // 设置事件监听器
    const setupEventListeners = () => {
      // 监听服务器状态更新
      const onServerStatusUpdate = (event) => {
        const { serverId: updatedServerId, data, timestamp } = event.detail

        if (updatedServerId === serverId.value) {
          // 更新本地服务器数据
          server.value = { ...server.value, ...data }
          lastUpdate.value = timestamp

          // 实时更新图表
          if (data.players_online !== undefined) {
            addDataPointToChart(data.players_online, timestamp)
          }
        }
      }

      // 监听玩家活动，同时更新图表
      const onPlayerActivity = (event) => {
        const { serverId: activityServerId, data, timestamp } = event.detail

        if (activityServerId === serverId.value && data.players_online !== undefined) {
          addDataPointToChart(data.players_online, timestamp)
        }
      }

      // 添加事件监听器
      window.addEventListener('server-status-update', onServerStatusUpdate)
      window.addEventListener('player-join', onPlayerActivity)
      window.addEventListener('player-leave', onPlayerActivity)

      // 保存清理函数
      eventCleanupFunctions.push(
        () => window.removeEventListener('server-status-update', onServerStatusUpdate),
        () => window.removeEventListener('player-join', onPlayerActivity),
        () => window.removeEventListener('player-leave', onPlayerActivity)
      )
    }

    const cleanupEventListeners = () => {
      eventCleanupFunctions.forEach(cleanup => cleanup())
      eventCleanupFunctions.length = 0
    }

    onMounted(async () => {
      await loadServer()
      await loadOnlinePlayers()
      setupEventListeners()

      // 初始化segmented-button的选中状态
      await nextTick(() => {
        if (segmentedButtonGroup.value) {
          // 设置初始选中状态为"实时"
          const realtimeButton = segmentedButtonGroup.value.querySelector('mdui-segmented-button[value="realtime"]')
          if (realtimeButton) {
            realtimeButton.selected = true
          }
        }
      })

      // 初始化实时模式，加载最近30分钟的数据
      if (isRealTimeMode.value) {
        await loadRecentDataForRealtime()
      } else {
        await loadServerStats()
      }
    })

    onUnmounted(() => {
      cleanupEventListeners()
    })

    return {
      server,
      lastUpdate,
      onlinePlayers,
      serverActivities,
      chartComponent,
      segmentedButtonGroup,
      chartData,
      chartOptions,
      chartKey,
      selectedTimeRange,
      isRealTimeMode,
      timeRanges,
      goBack,
      goToPlayer,
      formatTime,
      formatJoinTime,
      formatUpdateTime,
      formatDuration,
      getPingClass,
      selectTimeRange
    }
  }
}
</script>

<style scoped>
.server-detail {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.list {
  max-height: 320px;
  overflow-y: auto;
}

.server-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.server-title h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 500;
}

.server-address {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
  margin-top: 4px;
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 24px 8px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
}

.info-list {
  padding: 6px 24px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-item .label {
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
}

.info-item .value {
  color: rgb(var(--mdui-color-on-surface-variant));
  display: flex;
  align-items: center;
  gap: 8px;
}

.motd {
  text-align: right;
  max-width: 200px;
  word-break: break-word;
}

.player-progress {
  width: 60px;
  height: 4px;
}

.ping-good {
  color: rgb(var(--mdui-color-success));
}

.ping-fair {
  color: rgb(var(--mdui-color-warning));
}

.ping-poor {
  color: rgb(var(--mdui-color-error));
}

.ping-unknown {
  color: rgb(var(--mdui-color-on-surface-variant));
}

.player-info {
  flex: 1;
}

.player-name {
  font-size: 16px;
  font-weight: 500;
}

.player-rank {
  font-size: 14px;
  color: rgb(var(--mdui-color-primary));
}

.player-time {
  font-size: 12px;
  color: rgb(var(--mdui-color-on-surface-variant));
  text-align: right;
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

.activity-info {
  flex: 1;
}

.activity-text {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
  display: flex;
  align-items: center;
  gap: 4px;
}

.activity-time {
  font-size: 12px;
  color: rgb(var(--mdui-color-on-surface-variant));
  text-align: right;
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

.chart-card {
  width: 100%;
  margin-top: 24px;
}

.chart-container {
  height: 400px;
  padding: 24px;
}

.chart-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

@media (max-width: 768px) {
  .server-detail {
    padding: 16px;
  }

  .content-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .server-header {
    flex-wrap: wrap;
    gap: 12px;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .motd {
    text-align: left;
    max-width: none;
  }
}
</style>