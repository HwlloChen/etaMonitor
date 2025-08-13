import { defineStore } from 'pinia'
import axios from 'axios'

// 创建API实例
const api = axios.create({
  baseURL: '/api'
})

// 添加请求拦截器，自动添加token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token'),
    user: null,
    isAuthenticated: !!localStorage.getItem('token')
  }),

  actions: {
    async login(username, password) {
      try {
        const response = await api.post('/auth/login', { username, password })
        
        if (response.data.success) {
          const { token, user } = response.data
          
          this.token = token
          this.user = user
          this.isAuthenticated = true
          
          localStorage.setItem('token', token)
          
          return response.data
        } else {
          throw new Error(response.data.error?.message || '登录失败')
        }
      } catch (error) {
        console.error('Login error:', error)
        throw error.response?.data?.error?.message || error.message || '登录失败'
      }
    },

    async logout() {
      try {
        if (this.token) {
          await api.post('/auth/logout')
        }
      } catch (error) {
        console.error('Logout error:', error)
      } finally {
        this.token = null
        this.user = null
        this.isAuthenticated = false
        localStorage.removeItem('token')
      }
    },

    async refreshToken() {
      try {
        // 如果没有token，直接失败
        if (!this.token) {
          throw new Error('No token to refresh')
        }

        const response = await api.post('/auth/refresh')
        
        if (response.data.success) {
          this.token = response.data.token
          this.isAuthenticated = true
          localStorage.setItem('token', response.data.token)
          // 更新 axios 默认 headers
          api.defaults.headers.common['Authorization'] = `Bearer ${response.data.token}`
          return response.data
        } else {
          throw new Error('Token refresh failed')
        }
      } catch (error) {
        // 清理认证状态
        await this.logout()
        throw error
      }
    },

    checkAuth() {
      const token = localStorage.getItem('token')
      if (token) {
        this.token = token
        this.isAuthenticated = true
        // 可以在这里验证token有效性
        this.validateToken()
      }
    },

    async validateToken() {
      if (!this.token) return
      
      try {
        // 尝试获取用户信息来验证token
        const response = await api.get('/auth/me')
        if (response.data.success) {
          this.user = response.data.user
          this.isAuthenticated = true
        }
      } catch (error) {
        console.error('Token validation failed:', error)
        this.logout()
      }
    }
  }
})

// 添加响应拦截器，处理token过期 - 需要在store定义之后
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    const isRefreshRequest = originalRequest.url === '/auth/refresh'
    
    // 如果是刷新token的请求失败，直接退出
    if (isRefreshRequest) {
      localStorage.removeItem('token')
      // 重置所有认证状态
      const authStore = useAuthStore()
      await authStore.logout()
      return Promise.reject(error)
    }
    
    // 处理其他401错误
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true
      
      // 获取当前token
      const token = localStorage.getItem('token')
      if (!token) {
        return Promise.reject(error)
      }

      try {
        // 尝试刷新token
        const response = await api.post('/auth/refresh')
        if (response.data.success) {
          const newToken = response.data.token
          localStorage.setItem('token', newToken)
          
          // 更新当前store中的token
          const authStore = useAuthStore()
          authStore.token = newToken
          
          // 重新设置authorization header
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          // 为新请求设置新的token
          api.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
          return api(originalRequest)
        }
      } catch (refreshError) {
        // 刷新失败，执行登出操作
        const authStore = useAuthStore()
        await authStore.logout()
        // 抛出原始错误
        return Promise.reject(error)
      }
    }
    
    return Promise.reject(error)
  }
)

// 导出配置好的axios实例供其他store使用
export { api }