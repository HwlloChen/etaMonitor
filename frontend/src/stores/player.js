import { defineStore } from 'pinia'
import { api } from './auth'

export const usePlayerStore = defineStore('player', {
  state: () => ({
    player: {},
    sessions: [],
    stats: {},
    players: [],
    currentPlayer: null,
    recentActivities: [], // 最近的玩家活动
    onlinePlayers: new Map(), // serverId -> online players array
  }),

  actions: {
    async getPlayer(id) {
      try {
        const res = await api.get(`/players/${id}`)
        this.player = res.data.data
        this.currentPlayer = res.data.data
        return res.data
      } catch (error) {
        console.error('获取玩家详情失败:', error)
        throw error
      }
    },

    async getPlayerSessions(id) {
      try {
        const res = await api.get(`/players/${id}/sessions`)
        this.sessions = res.data.data
        return res.data
      } catch (error) {
        console.error('获取玩家会话失败:', error)
        throw error
      }
    },

    async getPlayerStats(id) {
      try {
        const res = await api.get(`/stats/players/${id}`)
        this.stats = res.data.data
        return res.data
      } catch (error) {
        console.error('获取玩家统计失败:', error)
        throw error
      }
    },

    async getPlayers(params = {}) {
      try {
        const response = await api.get('/players', { params })
        this.players = response.data.data || []
        return response.data
      } catch (error) {
        console.error('获取玩家列表失败:', error)
        throw error
      }
    },

    // 处理玩家加入事件
    handlePlayerJoin(data, serverId, timestamp) {
      const activity = {
        type: 'join',
        playerId: data.username,
        playerName: data.username,
        playerAvatar: data.avatar,
        playerRank: data.rank,
        serverId: serverId,
        serverName: data.server_name,
        timestamp: timestamp,
        playersOnline: data.players_online
      }

      // 添加到最近活动
      this.recentActivities.unshift(activity)
      
      // 保持最近活动列表不超过50条
      if (this.recentActivities.length > 50) {
        this.recentActivities = this.recentActivities.slice(0, 50)
      }

      // 更新在线玩家列表
      const onlinePlayers = this.onlinePlayers.get(serverId) || []
      const existingIndex = onlinePlayers.findIndex(p => p.username === data.username)
      
      if (existingIndex === -1) {
        onlinePlayers.push({
          username: data.username,
          uuid: data.uuid,
          avatar: data.avatar,
          rank: data.rank,
          joinTime: timestamp
        })
        this.onlinePlayers.set(serverId, onlinePlayers)
      }

      console.log(`玩家加入: ${data.username} -> ${data.server_name}`)
    },

    // 处理玩家离开事件
    handlePlayerLeave(data, serverId, timestamp) {
      const activity = {
        type: 'leave',
        playerId: data.username,
        playerName: data.username,
        playerAvatar: data.avatar,
        playerRank: data.rank,
        serverId: serverId,
        serverName: data.server_name,
        timestamp: timestamp,
        playersOnline: data.players_online,
        sessionDuration: data.session_duration
      }

      // 添加到最近活动
      this.recentActivities.unshift(activity)
      
      // 保持最近活动列表不超过50条
      if (this.recentActivities.length > 50) {
        this.recentActivities = this.recentActivities.slice(0, 50)
      }

      // 从在线玩家列表移除
      const onlinePlayers = this.onlinePlayers.get(serverId) || []
      const filteredPlayers = onlinePlayers.filter(p => p.username !== data.username)
      this.onlinePlayers.set(serverId, filteredPlayers)

      console.log(`玩家离开: ${data.username} -> ${data.server_name} (游戏时长: ${data.session_duration}秒)`)
    },

    // 从服务器获取在线玩家
    async fetchOnlinePlayers(serverId) {
      try {
        const res = await api.get(`/servers/${serverId}/players`)
        const players = res.data.data || []
        this.onlinePlayers.set(serverId, players)
        return res.data
      } catch (error) {
        console.error('获取在线玩家失败:', error)
        this.onlinePlayers.set(serverId, [])
        throw error
      }
    },

    // 获取服务器的在线玩家
    getOnlinePlayers(serverId) {
      return this.onlinePlayers.get(serverId) || []
    },

    // 清理在线玩家数据
    clearOnlinePlayers(serverId = null) {
      if (serverId) {
        this.onlinePlayers.delete(serverId)
      } else {
        this.onlinePlayers.clear()
      }
    },

    // 获取最近活动
    getRecentActivities(limit = 10) {
      return this.recentActivities.slice(0, limit)
    },

    // 清理最近活动
    clearRecentActivities() {
      this.recentActivities = []
    },

    // 格式化玩家会话时长
    formatDuration(seconds) {
      if (seconds < 60) {
        return `${seconds}秒`
      } else if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60)
        return `${minutes}分钟`
      } else {
        const hours = Math.floor(seconds / 3600)
        const minutes = Math.floor((seconds % 3600) / 60)
        return `${hours}小时${minutes}分钟`
      }
    }
  },

  getters: {
    totalOnlinePlayers() {
      let total = 0
      for (const players of this.onlinePlayers.values()) {
        total += players.length
      }
      return total
    },

    recentJoins() {
      return this.recentActivities
        .filter(activity => activity.type === 'join')
        .slice(0, 5)
    },

    recentLeaves() {
      return this.recentActivities
        .filter(activity => activity.type === 'leave')
        .slice(0, 5)
    }
  }
})
