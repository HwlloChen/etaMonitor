<template>
    <div class="servers">
        <div class="header">
            <h1>所有服务器</h1>
            <mdui-button variant="tonal" icon="refresh" @click="refreshServers" :loading="isRefreshing">
                刷新
            </mdui-button>
        </div>

        <!-- 初始加载状态 -->
        <div v-if="isInitialLoading" class="loading-state">
            <mdui-circular-progress></mdui-circular-progress>
            <p>加载服务器列表中...</p>
        </div>

        <!-- 服务器网格 -->
        <div v-else-if="servers.length > 0" class="servers-grid">
            <mdui-card 
                v-for="server in servers" 
                :key="server.id" 
                class="server-card" 
                @click="goToServer(server.id)"
            >
                <!-- 服务器信息 -->
                <div class="server-content">
                    <div class="server-title">
                        <div>
                            <h2>{{ server.name }}</h2>
                            <div class="server-address">{{ server.address }}:{{ server.port }}</div>
                        </div>
                        <mdui-chip :icon="server.status === 'online' ? 'wifi_tethering' : 'wifi_off'">
                            {{ server.status === 'online' ? '在线' : '离线' }}
                        </mdui-chip>
                    </div>

                    <!-- 服务器详细信息 -->
                    <div class="server-info">
                        <div class="info-item">
                            <mdui-icon name="people"></mdui-icon>
                            <span>{{ server.players_online || 0 }}/{{ server.max_players || 0 }} 玩家在线</span>
                        </div>
                        <div class="info-item" v-if="server.status === 'online'">
                            <mdui-icon name="speed"></mdui-icon>
                            <span>{{ server.ping || 0 }}ms</span>
                        </div>
                        <div class="info-item" v-if="server.motd">
                            <mdui-icon name="description"></mdui-icon>
                            <span class="motd">{{ server.motd }}</span>
                        </div>
                    </div>
                </div>

                <!-- 加载覆盖层（用于刷新时的视觉反馈） -->
                <div v-if="isRefreshing" class="server-loading-overlay">
                    <mdui-circular-progress></mdui-circular-progress>
                </div>
            </mdui-card>
        </div>

        <!-- 空状态 -->
        <div v-else class="empty-state">
            <mdui-icon name="dns_off" class="empty-icon"></mdui-icon>
            <p>暂无服务器</p>
            <mdui-button variant="text" @click="refreshServers" :loading="isRefreshing">
                重新加载
            </mdui-button>
        </div>
    </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useServerStore } from '../stores/server'

export default {
    name: 'Servers',
    setup() {
        const router = useRouter()
        const serverStore = useServerStore()
        const servers = ref([])
        const isRefreshing = ref(false)
        const isInitialLoading = ref(false)

        // 事件监听器清理函数
        const eventCleanupFunctions = []

        const loadServers = async (showInitialLoading = false) => {
            if (showInitialLoading) {
                isInitialLoading.value = true
            }

            try {
                const response = await serverStore.getServers()
                if (response && response.data) {
                    servers.value = response.data
                }
            } catch (error) {
                console.error('加载服务器列表失败:', error)
                // 可以在这里添加错误提示
            } finally {
                if (showInitialLoading) {
                    isInitialLoading.value = false
                }
            }
        }

        const refreshServers = async () => {
            if (isRefreshing.value) return
            
            isRefreshing.value = true
            try {
                // 添加最小加载时间以提供更好的用户反馈
                const loadPromise = loadServers(false)
                const minDelayPromise = new Promise(resolve => setTimeout(resolve, 500))
                
                await Promise.all([loadPromise, minDelayPromise])
            } catch (error) {
                console.error('刷新服务器列表失败:', error)
            } finally {
                isRefreshing.value = false
            }
        }

        const goToServer = (id) => {
            // 防止在刷新时误点击
            if (!isRefreshing.value) {
                router.push(`/server/${id}`)
            }
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
            }

            // 监听服务器连接状态变化
            const onServerConnectionChange = (event) => {
                const { serverId, status } = event.detail
                
                // 更新特定服务器的状态
                const serverIndex = servers.value.findIndex(s => s.id === serverId)
                if (serverIndex !== -1) {
                    servers.value[serverIndex] = {
                        ...servers.value[serverIndex],
                        status: status
                    }
                }
            }

            // 添加事件监听器
            window.addEventListener('server-status-update', onServerStatusUpdate)
            window.addEventListener('server-connection-change', onServerConnectionChange)

            // 保存清理函数
            eventCleanupFunctions.push(
                () => window.removeEventListener('server-status-update', onServerStatusUpdate),
                () => window.removeEventListener('server-connection-change', onServerConnectionChange)
            )
        }

        const cleanupEventListeners = () => {
            eventCleanupFunctions.forEach(cleanup => cleanup())
            eventCleanupFunctions.length = 0
        }

        onMounted(() => {
            // 初始加载时显示加载状态
            loadServers(true)
            setupEventListeners()
        })

        onUnmounted(() => {
            cleanupEventListeners()
        })

        return {
            servers,
            isRefreshing,
            isInitialLoading,
            refreshServers,
            goToServer
        }
    }
}
</script>

<style scoped>
.servers {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
}

.header h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 500;
}

.loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 24px;
    gap: 16px;
    color: rgb(var(--mdui-color-on-surface-variant));
}

.servers-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 24px;
    position: relative;
}

.server-card {
    cursor: pointer;
    box-shadow: var(--mdui-elevation-level1);
    transition: box-shadow 0.2s ease, opacity 0.2s ease;
    position: relative;
    overflow: hidden;
}

.server-card:hover:not(:has(.server-loading-overlay)) {
    box-shadow: var(--mdui-elevation-level3);
}

.server-card:has(.server-loading-overlay) {
    pointer-events: none;
    opacity: 0.7;
}

.server-header {
    padding: 16px 16px 0;
}

.server-content {
    padding: 16px;
    position: relative;
}

.server-title {
    margin-bottom: 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.server-title h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 500;
}

.server-address {
    font-size: 14px;
    color: rgb(var(--mdui-color-on-surface-variant));
    margin-top: 4px;
}

.server-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.info-item {
    display: flex;
    align-items: center;
    gap: 8px;
    color: rgb(var(--mdui-color-on-surface-variant));
    font-size: 14px;
}

.info-item :deep(mdui-icon) {
    font-size: 20px;
    color: rgb(var(--mdui-color-on-surface-variant));
}

.motd {
    flex: 1;
    word-break: break-word;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

.server-loading-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(var(--mdui-color-surface-container), 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1;
    backdrop-filter: blur(2px);
}

.empty-state {
    text-align: center;
    padding: 80px 24px;
    color: rgb(var(--mdui-color-on-surface-variant));
}

.empty-icon {
    font-size: 48px;
    opacity: 0.5;
    margin-bottom: 16px;
}

.empty-state p {
    margin-bottom: 24px;
    font-size: 16px;
}

@media (max-width: 768px) {
    .servers {
        padding: 16px;
    }

    .servers-grid {
        grid-template-columns: 1fr;
        gap: 16px;
    }

    .header {
        margin-bottom: 16px;
    }

    .loading-state {
        padding: 60px 16px;
    }

    .empty-state {
        padding: 60px 16px;
    }
}
</style>