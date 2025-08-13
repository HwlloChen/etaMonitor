import { defineStore } from 'pinia'
import { api } from './auth'

export const useServerStore = defineStore('server', {
  state: () => ({
    servers: [],
    currentServer: null,
    stats: null,
    realtimeData: new Map(), // serverId -> realtime data
  }),

  actions: {
    async getServers(params = {}) {
      try {
        const response = await api.get('/servers', { params })
        this.servers = response.data.data || []
        return response.data
      } catch (error) {
        console.error('获取服务器列表失败:', error)
        throw error
      }
    },

    async getServer(id) {
      try {
        const response = await api.get(`/servers/${id}`)
        this.currentServer = response.data.data
        return response.data
      } catch (error) {
        console.error('获取服务器详情失败:', error)
        throw error
      }
    },

    async createServer(serverData) {
      try {
        const response = await api.post('/servers', serverData)
        if (response.data.success) {
          // 添加到本地列表
          this.servers.push(response.data.data)
          return response.data
        } else {
          throw new Error(response.data.error?.message || '创建服务器失败')
        }
      } catch (error) {
        console.error('创建服务器失败:', error)
        throw error.response?.data?.error?.message || error.message || '创建服务器失败'
      }
    },

    async updateServer(id, serverData) {
      try {
        const response = await api.put(`/servers/${id}`, serverData)
        if (response.data.success) {
          // 更新本地列表
          const index = this.servers.findIndex(s => s.id === id)
          if (index !== -1) {
            this.servers[index] = response.data.data
          }
          return response.data
        } else {
          throw new Error(response.data.error?.message || '更新服务器失败')
        }
      } catch (error) {
        console.error('更新服务器失败:', error)
        throw error.response?.data?.error?.message || error.message || '更新服务器失败'
      }
    },

    async deleteServer(id) {
      try {
        const response = await api.delete(`/servers/${id}`)
        if (response.data.success) {
          // 从本地列表移除
          this.servers = this.servers.filter(s => s.id !== id)
          return response.data
        } else {
          throw new Error(response.data.error?.message || '删除服务器失败')
        }
      } catch (error) {
        console.error('删除服务器失败:', error)
        throw error.response?.data?.error?.message || error.message || '删除服务器失败'
      }
    },

    async pingServer(id) {
      try {
        const response = await api.post(`/servers/${id}/ping`)
        if (response.data.success) {
          // 更新本地服务器状态
          const index = this.servers.findIndex(s => s.id === id)
          if (index !== -1) {
            this.servers[index].status = 'online'
          }
          return response.data
        } else {
          throw new Error(response.data.error?.message || 'Ping服务器失败')
        }
      } catch (error) {
        console.error('Ping服务器失败:', error)
        throw error.response?.data?.error?.message || error.message || 'Ping服务器失败'
      }
    },

    async detectServer(address, port) {
      try {
        const response = await api.post('/servers/detect', { address, port })
        return response.data
      } catch (error) {
        console.error('检测服务器失败:', error)
        throw error.response?.data?.error?.message || error.message || '检测服务器失败'
      }
    },

    async getStats() {
      try {
        const response = await api.get('/stats/overview')
        this.stats = response.data.data
        return response.data
      } catch (error) {
        console.error('获取统计数据失败:', error)
        throw error
      }
    },

    async getServerStats(id, range = '24h') {
      try {
        const response = await api.get(`/stats/servers/${id}`, {
          params: { range }
        })
        return response.data
      } catch (error) {
        console.error('获取服务器统计失败:', error)
        throw error
      }
    },

    async getPlayers(params = {}) {
      try {
        const response = await api.get('/players', { params })
        return response.data
      } catch (error) {
        console.error('获取玩家列表失败:', error)
        throw error
      }
    },

    async getPlayer(id) {
      try {
        const response = await api.get(`/players/${id}`)
        return response.data
      } catch (error) {
        console.error('获取玩家详情失败:', error)
        throw error
      }
    },

    async getPlayerSessions(id, params = {}) {
      try {
        const response = await api.get(`/players/${id}/sessions`, { params })
        return response.data
      } catch (error) {
        console.error('获取玩家会话失败:', error)
        throw error
      }
    },

    // WebSocket实时数据更新方法
    updateServerRealtime(serverId, data) {
      this.realtimeData.set(serverId, {
        ...this.realtimeData.get(serverId),
        ...data,
        lastUpdate: new Date()
      })

      // 更新服务器列表中的对应数据
      const serverIndex = this.servers.findIndex(s => s.id === serverId)
      if (serverIndex !== -1) {
        this.servers[serverIndex] = {
          ...this.servers[serverIndex],
          ...data
        }
      }

      // 如果是当前查看的服务器，也更新currentServer
      if (this.currentServer && this.currentServer.id === serverId) {
        this.currentServer = {
          ...this.currentServer,
          ...data
        }
      }
    },

    // 获取服务器的实时数据
    getServerRealtime(serverId) {
      return this.realtimeData.get(serverId) || null
    },

    // 清理实时数据
    clearRealtimeData() {
      this.realtimeData.clear()
    }
  }
})