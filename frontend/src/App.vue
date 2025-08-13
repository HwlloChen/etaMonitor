<template>
  <div id="app">
    <mdui-layout class="app-layout">
      <!-- 顶部导航栏 -->
      <mdui-top-app-bar :variant="isCompact ? 'small' : 'center-aligned'" scroll-behavior="elevate">
        <!-- 左侧菜单按钮 (仅在移动端显示) -->
        <mdui-button-icon v-if="isMobile" icon="menu" @click="toggleDrawer"></mdui-button-icon>

        <!-- 应用标题 -->
        <mdui-top-app-bar-title>
          {{ pageTitle }}
        </mdui-top-app-bar-title>

        <!-- 右侧操作按钮 -->
        <div class="header-actions">
          <!-- WebSocket状态指示器 -->
          <mdui-chip :icon="wsStatusIcon" v-if="!isMobile">{{ wsStatusText }}</mdui-chip>
          <mdui-button-icon :icon="themeIcon" @click="toggleTheme"></mdui-button-icon>
        </div>
      </mdui-top-app-bar>

      <!-- 侧边导航 (NavigationRail for desktop, NavigationDrawer for mobile) -->
      <mdui-navigation-rail v-if="!isMobile" class="app-rail" :value="$route.path" :alignment="railAlignment">
        <mdui-navigation-rail-item v-for="item in navigationItems" :key="item.path" :icon="item.icon + '--outlined'"
          :active-icon="item.icon + '--filled'" :value="item.path" @click="handleNavClick(item.path)">
          {{ item.title }}
        </mdui-navigation-rail-item>

        <!-- 底部设置按钮 -->
        <div slot="bottom" class="rail-bottom">
          <mdui-button-icon v-if="isAuthenticated" icon="admin_panel_settings"
            @click="$router.push('/admin')"></mdui-button-icon>
          <mdui-button-icon icon="login" @click="showLoginDialog = !isAuthenticated"
            v-if="!isAuthenticated"></mdui-button-icon>
          <mdui-button-icon icon="logout" @click="handleLogout" v-else></mdui-button-icon>
        </div>
      </mdui-navigation-rail>

      <!-- 移动端抽屉导航 -->
      <mdui-navigation-drawer v-if="isMobile" class="app-drawer" placement="left" :modal="true" :open="isDrawerOpen"
        @close="onDrawerClose" @overlay-click="closeDrawer">
        <!-- 抽屉头部 -->
        <div class="drawer-header">
          <div class="app-info">
            <mdui-icon name="dns" class="app-logo-large"></mdui-icon>
            <div class="app-details">
              <h2>etaMonitor</h2>
              <p>Minecraft服务器监控系统</p>
            </div>
          </div>
        </div>

        <mdui-divider></mdui-divider>

        <!-- 导航列表 -->
        <mdui-list class="drawer-nav">
          <mdui-list-item v-for="item in navigationItems" :key="item.path" :active="$route.path === item.path"
            @click="handleNavClick(item.path)" :icon="item.icon" :headline="item.title">
          </mdui-list-item>
        </mdui-list>

        <mdui-divider></mdui-divider>

        <!-- 底部认证区域 -->
        <mdui-list class="drawer-footer">
          <mdui-list-item v-if="isAuthenticated" @click="$router.push('/admin')" headline="管理面板"
            icon="admin_panel_settings"></mdui-list-item>

          <mdui-list-item @click="showLoginDialog = !isAuthenticated" v-if="!isAuthenticated" headline="管理员登录"
            icon="login"></mdui-list-item>

          <mdui-list-item @click="handleLogout" v-else headline="退出登录" icon="logout"></mdui-list-item>
        </mdui-list>
      </mdui-navigation-drawer>

      <!-- 主内容区域 -->
      <mdui-layout-main class="app-main">
        <div class="main-content" :class="{ 'with-rail': !isMobile }">
          <!-- 页面内容 -->
          <div class="page-content">
            <router-view />
          </div>
        </div>
      </mdui-layout-main>
    </mdui-layout>

    <!-- 登录对话框 -->
    <mdui-dialog :open="showLoginDialog" @close="onLoginDialogClose" headline="管理员登录" class="login-dialog">
      <div class="login-form">
        <mdui-text-field label="用户名" :value="loginForm.username" @input="loginForm.username = $event.target.value"
          icon="person" required @keydown.enter="$refs.passwordField.focus()"></mdui-text-field>

        <mdui-text-field ref="passwordField" label="密码" type="password" :value="loginForm.password"
          @input="loginForm.password = $event.target.value" icon="lock" required
          @keydown.enter="handleLogin"></mdui-text-field>

        <div v-if="loginError" class="error-message">
          <mdui-icon name="error" class="error-icon"></mdui-icon>
          {{ loginError }}
        </div>
      </div>

      <mdui-button slot="action" variant="text" @click="onLoginDialogClose">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" @click="handleLogin" :loading="isLoggingIn"
        :disabled="!loginForm.username || !loginForm.password">
        登录
      </mdui-button>
    </mdui-dialog>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { getWebSocketStatus, getWebSocketStatusText } from './utils/ws'
import { snackbar, getTheme, setTheme } from 'mdui'

export default {
  name: 'App',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const authStore = useAuthStore()

    // 响应式状态
    const windowWidth = ref(window.innerWidth)
    const isDrawerOpen = ref(false)
    const showLoginDialog = ref(false)
    const isLoggingIn = ref(false)
    const wsStatus = ref('disconnected')
    const wsStatusText = ref('连接断开')

    // 登录表单
    const loginForm = ref({
      username: '',
      password: ''
    })

    // 主题颜色
    const themeIcon = ref(['brightness_auto', 'light_mode', 'dark_mode'][['auto', 'light', 'dark'].indexOf(getTheme())])

    // 计算属性
    const isMobile = computed(() => windowWidth.value < 960)
    const isCompact = computed(() => windowWidth.value < 1200)
    const railAlignment = computed(() => isCompact.value ? 'start' : 'center')
    const isAuthenticated = computed(() => authStore.isAuthenticated)

    // WebSocket状态相关计算属性
    const wsStatusIcon = computed(() => {
      switch (wsStatus.value) {
        case 'connecting':
          return 'sync'
        case 'connected':
          return 'link'
        case 'closing':
          return 'sync_disabled'
        case 'closed':
        case 'disconnected':
          return 'link_off'
        default:
          return 'link_off'
      }
    })

    const wsStatusColor = computed(() => {
      switch (wsStatus.value) {
        case 'connecting':
          return 'warning'
        case 'connected':
          return 'success'
        default:
          return 'error'
      }
    })

    // 导航菜单项
    const navigationItems = [
      {
        path: '/',
        title: '首页',
        icon: 'home'
      },
      {
        path: '/servers',
        title: '服务器',
        icon: 'dns'
      }
    ]

    // 页面标题
    const pageTitle = computed(() => {
      const currentItem = navigationItems.find(item => item.path === route.path)
      if (currentItem) {
        return isMobile.value ? currentItem.title : `etaMonitor - ${currentItem.title}`
      }
      return 'etaMonitor'
    })

    // 抽屉方法
    const toggleDrawer = () => {
      if (isMobile.value) {
        isDrawerOpen.value = !isDrawerOpen.value
      }
    }

    const closeDrawer = () => {
      if (isMobile.value) {
        isDrawerOpen.value = false
      }
    }

    const onDrawerClose = () => {
      isDrawerOpen.value = false
    }

    // 导航处理
    const handleNavClick = (path) => {
      router.push(path)
      if (isMobile.value) {
        closeDrawer()
      }
    }

    // 主题切换
    const toggleTheme = () => {
      switch (getTheme()) {
        case 'auto':
          setTheme('dark');
          break;
        case 'dark':
          setTheme('light');
          break;
        default:
          setTheme('auto');
      }
      localStorage.setItem('theme', getTheme())
      themeIcon.value = ['brightness_auto', 'light_mode', 'dark_mode'][['auto', 'light', 'dark'].indexOf(getTheme())]
    }

    // 登录处理
    const loginError = ref('')

    const handleLogin = async () => {
      if (!loginForm.value.username || !loginForm.value.password) {
        loginError.value = '请输入用户名和密码'
        return
      }

      loginError.value = ''
      isLoggingIn.value = true

      try {
        await authStore.login(loginForm.value.username, loginForm.value.password)
        showLoginDialog.value = false
        snackbar({
          message: '登录成功',
          placement: 'bottom-end'
        })
        setTimeout(() => {
          loginForm.value = { username: '', password: '' }
        }, 600);
      } catch (error) {
        console.error('登录失败:', error)
        loginError.value = error
      } finally {
        isLoggingIn.value = false
      }
    }

    const onLoginDialogClose = () => {
      showLoginDialog.value = false
      loginError.value = ''
      loginForm.value = { username: '', password: '' }
    }

    // 退出登录
    const handleLogout = async () => {
      try {
        await authStore.logout()
        snackbar({
          message: '已退出登录',
          placement: 'bottom-end'
        })
      } catch (error) {
        console.error('退出登录失败:', error)
        // 直接清除localStorage.token.
        localStorage.removeItem('token')
      }
    }

    // WebSocket状态更新
    const updateWebSocketStatus = () => {
      wsStatus.value = getWebSocketStatus()
      wsStatusText.value = getWebSocketStatusText()
    }

    // WebSocket操作处理
    const handleWebSocketAction = () => {
      // 这里可以添加重连逻辑或显示详细状态
    }

    // 窗口大小变化处理
    const handleResize = () => {
      const newWidth = window.innerWidth
      const wasMobile = windowWidth.value < 960
      const isMobileNow = newWidth < 960

      windowWidth.value = newWidth

      if (wasMobile && !isMobileNow) {
        closeDrawer()
      }
    }

    // 监听路由变化，移动端时关闭抽屉
    watch(route, () => {
      if (isMobile.value) {
        closeDrawer()
      }
    })

    // 生命周期
    onMounted(() => {
      window.addEventListener('resize', handleResize)
      authStore.checkAuth()

      // 开始WebSocket状态监听
      updateWebSocketStatus()
      const statusTimer = setInterval(updateWebSocketStatus, 1000)

      // 保存定时器以便清理
      window.wsStatusTimer = statusTimer
    })

    onUnmounted(() => {
      window.removeEventListener('resize', handleResize)

      // 清理状态监听定时器
      if (window.wsStatusTimer) {
        clearInterval(window.wsStatusTimer)
        window.wsStatusTimer = null
      }
    })

    return {
      // reactive data
      isDrawerOpen,
      showLoginDialog,
      isLoggingIn,
      loginForm,
      wsStatus,
      wsStatusText,
      themeIcon,

      // computed
      isMobile,
      isCompact,
      railAlignment,
      pageTitle,

      navigationItems,
      isAuthenticated,
      wsStatusIcon,
      wsStatusColor,

      // methods
      toggleDrawer,
      closeDrawer,
      onDrawerClose,
      handleNavClick,
      toggleTheme,
      handleLogin,
      handleLogout,
      handleWebSocketAction,
      loginError,
      onLoginDialogClose
    }
  }
}
</script>

<style lang="less" scoped>
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgb(var(--mdui-color-error));
  font-size: 0.875rem;
  padding: 4px 0;
}

.error-icon {
  font-size: 18px;
  color: rgb(var(--mdui-color-error));
}

/* 全局布局 */
#app {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.app-layout {
  height: 100%;
  width: 100%;
  background-color: rgb(var(--mdui-color-surface));
  overflow: hidden;
}

.app-main {
  height: 100%;
  overflow: hidden;
}

.main-content {
  background-color: rgb(var(--mdui-color-surface-container));
  padding: 1rem;
  margin: 0 1rem 1rem 0.5rem;
  height: calc(100% - 2rem);
  overflow: auto;
  border-radius: 1.5rem;
}

.page-content {
  height: 100%;
  overflow: auto;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
}

/* 导航边栏 */
.app-rail {
  z-index: 1000;
  background-color: rgb(var(--mdui-color-surface));
}

.rail-bottom {
  padding: 16px;
  border-top: 1px solid rgb(var(--mdui-color-outline-variant));
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.drawer-header {
  padding: 24px 16px 16px;
  background: linear-gradient(135deg, rgb(var(--mdui-color-primary-container)), rgb(var(--mdui-color-secondary-container)));
}

.app-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.app-logo-large {
  font-size: 48px;
  color: rgb(var(--mdui-color-on-primary-container));
}

.app-details h2 {
  margin: 0 0 4px 0;
  font-size: 1.5rem;
  font-weight: 500;
  color: rgb(var(--mdui-color-on-primary-container));
}

.app-details p {
  margin: 0;
  font-size: 0.875rem;
  color: rgb(var(--mdui-color-on-primary-container));
  opacity: 0.8;
}

.drawer-nav {
  padding: 8px 0;
}

.drawer-footer {
  margin-top: auto;
  padding: 8px 0;
  border-top: 1px solid rgb(var(--mdui-color-outline-variant));
}

/* 登录对话框 */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0 24px;
  min-width: 280px;
}

/* 响应式设计 */
@media (max-width: 600px) {
  .main-content {
    padding: 0.5rem;
    margin: 0;
    height: calc(100% - 64px - 1rem);
    border-radius: 0;
  }

  .page-content {
    padding: 0;
  }

  .breadcrumb-nav {
    margin-bottom: 16px;
  }
}

/* 动画效果 */
.page-content {
  transition: all 0.3s var(--mdui-easing-standard);
}

.main-content {
  transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
</style>