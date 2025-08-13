<template>
    <div class="player-detail">
        <div class="player-header">
            <mdui-button variant="text" icon="arrow_back" @click="goBack">返回</mdui-button>
            <div class="player-title">
                <div class="player-name">
                    <mdui-avatar v-if="player.avatar" :src="player.avatar"></mdui-avatar>
                    <div class="name-badge">
                        <h1>{{ player.username || '加载中...' }}</h1>
                        <mdui-badge :style="{
                            backgroundColor: isOnline ? 'green' : 'grey'
                        }">
                            {{ isOnline ? '在线' : '离线' }}</mdui-badge>
                    </div>
                </div>
                <div class="player-uuid">UUID: {{ player.uuid || '-' }}</div>
            </div>
            <div class="session-filter">
                <span>服务器：</span>
                <mdui-select 
                    multiple 
                    clearable 
                    placeholder="全部" 
                    style="min-width: 180px"
                    @change="handleMduiSelectChange"
                    :disabled="isLoadingChart"
                    key="server-selector"
                    ref="serverSelectRef"
                >
                    <mdui-menu-item v-for="s in serverList" :key="s.id" :value="s.id">{{ s.name }}</mdui-menu-item>
                </mdui-select>
            </div>
        </div>
        <div class="content-grid">
            <mdui-card class="info-card">
                <div class="card-header">
                    <h2>玩家信息</h2>
                    <mdui-chip>{{ ratingTitle }}</mdui-chip>
                </div>
                <div class="info-list">
                    <div class="info-item"><span class="label">首次加入:</span><span class="value">{{
                        formatDateTime(player.first_seen || null) }}</span></div>
                    <div class="info-item"><span class="label">最近在线:</span><span class="value">{{
                        formatDateTime(player.last_seen || null) }}</span></div>
                    <div class="info-item"><span class="label">总在线时长:</span><span class="value">{{
                        formatDuration(totalPlaytime || 0) }}</span></div>
                    <div class="info-item"><span class="label">称号:</span><span class="value">{{ title || '无' }}</span>
                    </div>
                </div>
            </mdui-card>
            <mdui-card class="history-card">
                <div class="card-header">
                    <h2>在线历史</h2>
                </div>
                <div style="max-height: 320px; overflow-y: auto;">
                    <mdui-list v-if="filteredSessions.length > 0">
                        <mdui-list-item v-for="session in filteredSessions" :key="session.id">
                            <div class="session-info">
                                <div>服务器: {{ session.server?.name || session.server_id }}</div>
                                <div>加入: {{ formatDateTime(session.join_time) }}</div>
                                <div>离开: {{ session.leave_time ? formatDateTime(session.leave_time) : '在线中' }}</div>
                                <div>时长: {{ session.leave_time ? formatDuration(session.duration) :
                                    formatDuration(getCurrentSessionDuration(session)) }}</div>
                            </div>
                        </mdui-list-item>
                    </mdui-list>
                    <div v-else class="empty-state">
                        <mdui-icon name="history" style="font-size: 48px; opacity: 0.5;"></mdui-icon>
                        <p>暂无历史记录</p>
                    </div>
                </div>
            </mdui-card>
        </div>
        <mdui-card class="chart-card">
            <div class="card-header">
                <h2>在线时段分布</h2>
            </div>
            <div class="chart-container">
                <PlayerTimeChart ref="chartComponent" :chart-data="chartData" :chart-options="chartOptions" />
            </div>
        </mdui-card>
    </div>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePlayerStore } from '../stores/player'
import PlayerTimeChart from '../components/PlayerTimeChart.vue'

export default {
    name: 'PlayerDetail',
    components: {
        PlayerTimeChart
    },
    setup() {
        const route = useRoute()
        const router = useRouter()
        const playerStore = usePlayerStore()
        const player = ref({
            username: '',
            uuid: '',
            first_seen: null,
            last_seen: null,
            total_playtime: 0
        })
        const sessions = ref([])
        const chartComponent = ref(null)
        const serverSelectRef = ref(null)
        const ratingTitle = ref('')
        const title = ref('')
        const serverList = ref([])
        const selectedServerIds = ref([])
        const totalPlaytime = ref(0)
        const chartKey = ref(0)
        const isLoadingChart = ref(false)
        const chartColor = `rgb(${getComputedStyle(document.documentElement).getPropertyValue('--mdui-color-primary').trim()})`
        
        // 处理mdui-select的值变化
        const handleMduiSelectChange = (event) => {
            const newValue = event.target.value
            selectedServerIds.value = Array.isArray(newValue) ? newValue : []
            handleServerSelectionChange()
        }

        // 图表数据
        const chartData = ref({
            labels: [],
            datasets: [{
                label: '在线时段分布',
                data: [],
                backgroundColor: chartColor,
                borderColor: chartColor,
                borderWidth: 1
            }]
        })

        // 图表配置
        const chartOptions = ref({
            responsive: true,
            maintainAspectRatio: false,
            animation: true,
            scales: {
                x: {
                    title: { display: true, text: '小时段' },
                    ticks: {
                        maxRotation: 0,
                        autoSkip: true,
                        maxTicksLimit: 12
                    }
                },
                y: {
                    title: { display: true, text: '上线次数' },
                    beginAtZero: true,
                    ticks: {
                        stepSize: 1,
                        precision: 0
                    }
                }
            },
            plugins: {
                legend: {
                    position: 'top'
                },
                tooltip: {
                    callbacks: {
                        label: function (context) {
                            return `上线次数: ${context.parsed.y} 次`
                        }
                    }
                }
            }
        })

        const isOnline = computed(() => {
            return sessions.value.some(s => !s.leave_time)
        })

        // 获取所有服务器列表
        const loadServerList = async () => {
            try {
                const res = await fetch('/api/servers/')
                
                // 检查响应状态
                if (res.status === 404) {
                    router.push('/404')
                    return
                }
                
                const result = await res.json()
                if (result.success && Array.isArray(result.data)) {
                    serverList.value = result.data
                }
            } catch (error) {
                console.error('加载服务器列表失败:', error)
            }
        }

        // 过滤后的会话
        const filteredSessions = computed(() => {
            if (!selectedServerIds.value || selectedServerIds.value.length === 0) return sessions.value
            return sessions.value.filter(s => selectedServerIds.value.includes(String(s.server_id)) || selectedServerIds.value.includes(s.server_id))
        })

        // 构建API URL，包含服务器筛选参数
        const buildStatsUrl = () => {
            let url = `/api/stats/players/${route.params.id}`
            if (selectedServerIds.value && selectedServerIds.value.length > 0) {
                const params = selectedServerIds.value.map(id => `server_id=${id}`).join('&')
                url += `?${params}`
            }
            return url
        }

        // 加载图表数据
        const loadChartData = async () => {
            if (isLoadingChart.value) return // 防止重复加载
            
            try {
                isLoadingChart.value = true
                const url = buildStatsUrl()
                
                const res = await fetch(url)
                
                // 检查响应状态
                if (res.status === 404) {
                    router.push('/404')
                    return
                }
                
                const result = await res.json()
                
                const raw = (result.data && result.data.time_distribution) || {}

                const hours = Array.from({ length: 24 }, (_, i) => i)
                const labels = hours.map(h => `${h.toString().padStart(2, '0')}:00~${(h + 1) % 24 === 0 ? '00' : (h + 1).toString().padStart(2, '0')}:00`)
                const values = hours.map(h => raw[h] || 0)
                

                updateChart(labels, values)
            } catch (error) {
                console.error('加载图表数据失败:', error)
            } finally {
                isLoadingChart.value = false
            }
        }

        // 防抖的图表加载函数
        const debouncedLoadChart = debounce(loadChartData, 300)

        // 服务器选择变化处理
        const handleServerSelectionChange = () => {
            debouncedLoadChart()
        }

        const loadPlayerInfo = async () => {
            try {
                const res = await playerStore.getPlayer(route.params.id)
                if (res && res.data) {
                    player.value = res.data
                    ratingTitle.value = getRating(res.data.total_playtime)
                    title.value = getTitle(res.data.time_segments)
                } else {
                    console.error('玩家数据为空')
                }
            } catch (error) {
                console.error('加载玩家信息失败:', error)
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

        const loadSessions = async () => {
            try {
                const res = await playerStore.getPlayerSessions(route.params.id)
                sessions.value = res.data || []
                let sum = 0
                for (const s of sessions.value) {
                    sum += s.duration || 0
                }
                totalPlaytime.value = sum > 0 ? sum : (player.value.total_playtime || 0)
            } catch (error) {
                console.error('加载玩家会话失败:', error)
                // 检查是否为404错误
                if (error.response && error.response.status === 404) {
                    router.push('/404')
                    return
                }
                sessions.value = []
            }
        }

        // 防抖工具
        function debounce(fn, delay = 300) {
            let timer = null
            let pending = false

            const cleanup = () => {
                if (timer) {
                    clearTimeout(timer)
                    timer = null
                }
                pending = false
            }

            return async function (...args) {
                cleanup()

                return new Promise((resolve, reject) => {
                    pending = true
                    timer = setTimeout(async () => {
                        try {
                            if (pending) {
                                const result = await fn.apply(this, args)
                                resolve(result)
                            }
                        } catch (err) {
                            reject(err)
                        } finally {
                            cleanup()
                        }
                    }, delay)
                })
            }
        }

        // 计算"在线中"会话的当前时长
        const getCurrentSessionDuration = (session) => {
            if (!session.join_time || session.leave_time) return session.duration || 0
            const join = new Date(session.join_time).getTime()
            const now = Date.now()
            return Math.floor((now - join) / 1000)
        }

        const updateChart = (labels, values) => {
            
            // 更新图表数据 - 不强制重新渲染，让vue-chartjs处理
            chartData.value = {
                labels: [...labels],
                datasets: [{
                    label: '在线时段分布',
                    data: [...values],
                    backgroundColor: chartColor,
                    borderColor: chartColor,
                    borderWidth: 1
                }]
            }
            
        }

        const goBack = () => router.go(-1)
        const formatDate = ts => ts ? new Date(ts).toLocaleDateString('zh-CN') : '-'
        const formatDateTime = ts => ts ? new Date(ts).toLocaleString('zh-CN') : '-'
        const formatDuration = sec => {
            if (!sec) return '0分钟'
            const h = Math.floor(sec / 3600)
            const m = Math.floor((sec % 3600) / 60)
            return h > 0 ? `${h}小时${m}分钟` : `${m}分钟`
        }

        // 评级和称号逻辑可根据实际业务调整
        const getRating = (playtime) => {
            if (playtime > 100 * 3600) return '传奇玩家'
            if (playtime > 50 * 3600) return '骨灰玩家'
            if (playtime > 20 * 3600) return '活跃玩家'
            if (playtime > 5 * 3600) return '新锐玩家'
            return '新手'
        }

        const getTitle = (segments) => {
            if (!segments) return '无'
            if (segments.night > segments.day && segments.night > segments.evening) return '夜猫子'
            if (segments.evening > segments.day) return '黄昏勇士'
            if (segments.day > 0) return '白天达人'
            return '无'
        }

        // WebSocket事件监听器清理函数
        const eventCleanupFunctions = []

        // 设置WebSocket事件监听器
        const setupEventListeners = () => {
            const currentPlayerId = route.params.id

            // 监听玩家加入事件
            const onPlayerJoin = (event) => {
                const { data } = event.detail
                if (data.username === currentPlayerId || data.uuid === currentPlayerId) {
                    loadSessions()
                }
            }

            // 监听玩家离开事件
            const onPlayerLeave = (event) => {
                const { data } = event.detail
                if (data.username === currentPlayerId || data.uuid === currentPlayerId) {
                    loadSessions()
                    // 玩家状态变化时重新加载图表
                    debouncedLoadChart()
                }
            }

            // 添加事件监听器
            window.addEventListener('player-join', onPlayerJoin)
            window.addEventListener('player-leave', onPlayerLeave)

            // 保存清理函数
            eventCleanupFunctions.push(
                () => window.removeEventListener('player-join', onPlayerJoin),
                () => window.removeEventListener('player-leave', onPlayerLeave)
            )
        }

        // 清理事件监听器
        const cleanupEventListeners = () => {
            eventCleanupFunctions.forEach(cleanup => cleanup())
            eventCleanupFunctions.length = 0
        }

        onMounted(async () => {
            await loadPlayerInfo()
            await loadSessions()
            await loadServerList()
            setupEventListeners()
            await loadChartData()
        })

        onUnmounted(() => {
            cleanupEventListeners()
        })

        return {
            player,
            sessions,
            chartComponent,
            serverSelectRef,
            chartData,
            chartOptions,
            chartKey,
            ratingTitle,
            title,
            goBack,
            formatDate,
            formatDateTime,
            formatDuration,
            totalPlaytime,
            getCurrentSessionDuration,
            serverList,
            selectedServerIds,
            filteredSessions,
            handleServerSelectionChange,
            handleMduiSelectChange,
            isOnline,
            isLoadingChart
        }
    }
}
</script>

<style scoped>
.session-filter {
    margin-bottom: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    white-space: nowrap;
    font-size: 1.1rem;
    margin-left: auto;
    justify-content: flex-end;
}

.player-detail {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.player-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
}

.player-title {
    flex: 1;
}

.player-name {
    display: flex;
    align-items: center;
    gap: 16px;
}

:deep(mdui-avatar) {
    border-radius: 8px;
    overflow: hidden;
    width: 48px;
    height: 48px;
}

:deep(mdui-avatar img) {
    image-rendering: pixelated;
}

.name-badge {
    display: flex;
    align-items: center;
    gap: 8px;
}

.player-name h1 {
    margin: 0;
    font-size: 28px;
    font-weight: 500;
}

.player-uuid {
    font-size: 16px;
    color: rgb(var(--mdui-color-on-surface-variant));
    margin-top: 4px;
}

.content-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 24px;
    margin-bottom: 24px;
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

.info-list {
    padding: 0 24px 24px;
}

.info-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
}

.info-item:last-child {
    border-bottom: none;
}

.label {
    font-weight: 500;
    color: rgb(var(--mdui-color-on-surface-variant));
}

.value {
    font-weight: 500;
    color: rgb(var(--mdui-color-on-surface));
}

.history-card {
    max-height: 400px;
    overflow-y: auto;
}

.empty-state {
    text-align: center;
    padding: 48px 24px;
    color: rgb(var(--mdui-color-on-surface-variant));
}

.chart-card {
    margin-bottom: 24px;
    width: 100%;
}

.chart-container {
    height: 300px;
    padding: 24px;
}

.session-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

@media (max-width: 768px) {
    .content-grid {
        grid-template-columns: 1fr;
    }

    .player-header {
        flex-wrap: wrap;
    }

    .player-detail {
        padding: 16px;
    }
}
</style>