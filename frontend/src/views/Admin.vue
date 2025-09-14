<template>
  <div class="admin">
    <div class="admin-header">
      <h1>管理面板</h1>
      <mdui-button variant="filled" icon="add" @click="showAddDialog = true">
        添加服务器
      </mdui-button>
    </div>

    <!-- 服务器管理 -->
    <mdui-card class="servers-table">
      <div class="card-header">
        <h2>服务器管理</h2>
        <mdui-text-field placeholder="搜索服务器..." icon="search" :value="searchText"
          @input="searchText = $event.target.value"></mdui-text-field>
      </div>

      <div class="servers-table-container">
        <!-- 表格头部 -->
        <div class="table-header">
          <div class="header-row">
            <div class="header-cell name-col">名称</div>
            <div class="header-cell address-col">地址</div>
            <div class="header-cell type-col">类型</div>
            <div class="header-cell status-col">状态</div>
            <div class="header-cell players-col">在线人数</div>
            <div class="header-cell actions-col">操作</div>
          </div>
        </div>

        <!-- 表格内容 -->
        <div class="table-body">
          <div v-if="filteredServers.length === 0" class="empty-state">
            <mdui-icon name="dns" class="empty-icon"></mdui-icon>
            <p>暂无服务器数据</p>
            <mdui-button variant="outlined" @click="showAddDialog = true">
              添加第一个服务器
            </mdui-button>
          </div>

          <div v-for="server in filteredServers" :key="server.id" class="table-row"
            @click="$router.push(`/server/${server.id}`)">
            <div class="table-cell name-col">
              <div class="server-name">{{ server.name }}</div>
            </div>
            <div class="table-cell address-col">
              <div class="server-address">{{ server.address }}:{{ server.port }}</div>
            </div>
            <div class="table-cell type-col">
              <mdui-chip variant="outlined">
                {{ server.type === 'java' ? 'Java版' : server.type === 'bedrock' ? '基岩版' : '自动检测' }}
              </mdui-chip>
            </div>
            <div class="table-cell status-col">
              <mdui-chip :color="server.status === 'online' ? 'primary' : 'error'" variant="filled">
                <mdui-icon :name="server.status === 'online' ? 'check_circle' : 'error'" slot="icon"></mdui-icon>
                {{ server.status === 'online' ? '在线' : '离线' }}
              </mdui-chip>
            </div>
            <div class="table-cell players-col">
              <span class="player-count">
                {{ server.players_online || 0 }}/{{ server.max_players || 0 }}
              </span>
            </div>
            <div class="table-cell actions-col" @click.stop>
              <div class="action-buttons">
                <mdui-button-icon icon="edit" @click="editServer(server)" variant="standard"></mdui-button-icon>
                <mdui-button-icon icon="refresh" @click="pingServer(server)" variant="standard"></mdui-button-icon>
                <mdui-button-icon icon="delete" @click="deleteServer(server)" variant="standard"></mdui-button-icon>
              </div>
            </div>
          </div>
        </div>
      </div>
    </mdui-card>

    <!-- 数据库管理 -->
    <mdui-card class="database-management">
      <div class="card-header">
        <h2>数据库管理</h2>
      </div>

      <!-- 数据库统计信息 -->
      <div class="database-stats" v-if="databaseStats">
        <div class="stats-grid">
          <div class="stat-item">
            <mdui-icon name="dns" class="stat-icon"></mdui-icon>
            <div class="stat-info">
              <div class="stat-number">{{ databaseStats.servers }}</div>
              <div class="stat-label">服务器</div>
            </div>
          </div>
          <div class="stat-item">
            <mdui-icon name="people" class="stat-icon"></mdui-icon>
            <div class="stat-info">
              <div class="stat-number">{{ databaseStats.players }}</div>
              <div class="stat-label">玩家</div>
            </div>
          </div>
          <div class="stat-item">
            <mdui-icon name="timeline" class="stat-icon"></mdui-icon>
            <div class="stat-info">
              <div class="stat-number">{{ databaseStats.server_stats }}</div>
              <div class="stat-label">服务器统计</div>
            </div>
          </div>
          <div class="stat-item">
            <mdui-icon name="history" class="stat-icon"></mdui-icon>
            <div class="stat-info">
              <div class="stat-number">{{ databaseStats.player_sessions }}</div>
              <div class="stat-label">玩家会话</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 数据库操作 -->
      <div class="database-actions">
        <div class="action-group">
          <h3>数据库优化</h3>
          <p>清理冗余数据，减少数据库大小，同时保持数据采样的有效性</p>
          <mdui-button variant="filled" icon="tune" @click="optimizeDatabase" :loading="isOptimizing">
            {{ isOptimizing ? '正在优化...' : '优化数据库' }}
          </mdui-button>
        </div>

        <div class="action-group">
          <h3>数据备份</h3>
          <p>创建数据库备份，包含完整的数据和元数据信息</p>
          <div class="backup-controls">
            <mdui-button variant="filled" icon="backup" @click="createBackup" :loading="isBackingUp">
              {{ isBackingUp ? '正在备份...' : '创建备份' }}
            </mdui-button>
            <mdui-button variant="outlined" icon="list" @click="openBackupsDialog">
              管理备份
            </mdui-button>
          </div>
        </div>
      </div>
    </mdui-card>

    <!-- 备份管理对话框 -->
    <mdui-dialog :open="showBackupsDialog" @close="showBackupsDialog = false" headline="备份管理"
      style="--mdui-comp-dialog-body-width: 800px;">
      <div class="backup-content">
        <div class="backup-header">
          <mdui-button variant="filled" icon="refresh" @click="loadBackups" :loading="isLoadingBackups">
            刷新列表
          </mdui-button>
          <mdui-button variant="outlined" icon="cleaning_services" @click="cleanupBackups" :loading="isCleaningUp">
            {{ isCleaningUp ? '正在清理...' : '清理旧备份' }}
          </mdui-button>
        </div>

        <div class="backup-list" v-if="backups.length > 0">
          <div v-for="backup in backups" :key="backup.path" class="backup-item">
            <div class="backup-info">
              <div class="backup-name">{{ backup.name }}</div>
              <div class="backup-details">
                <span>大小: {{ formatFileSize(backup.size) }}</span>
                <span>创建时间: {{ formatDate(backup.created_time) }}</span>
              </div>
            </div>
            <div class="backup-actions">
              <mdui-button-icon icon="restore" @click="restoreBackup(backup)" variant="standard"></mdui-button-icon>
              <mdui-button-icon icon="delete" @click="deleteBackup(backup)" variant="standard"></mdui-button-icon>
            </div>
          </div>
        </div>

        <div v-else class="empty-backups">
          <mdui-icon name="backup" class="empty-icon"></mdui-icon>
          <p>暂无备份文件</p>
        </div>
      </div>

      <mdui-button slot="action" variant="text" @click="showBackupsDialog = false">
        关闭
      </mdui-button>
    </mdui-dialog>

    <!-- 确认恢复对话框 -->
    <mdui-dialog :open="showRestoreDialog" @close="showRestoreDialog = false" headline="确认恢复备份">
      <div>
        <p><strong>警告：</strong>恢复备份将会替换当前数据库，此操作不可逆！</p>
        <p>备份文件: {{ restoringBackup?.name }}</p>
        <p>当前数据库将自动备份以防数据丢失</p>
      </div>

      <mdui-button slot="action" variant="text" @click="showRestoreDialog = false">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" color="error" @click="confirmRestore">
        确认恢复
      </mdui-button>
    </mdui-dialog>

    <!-- 确认删除备份对话框 -->
    <mdui-dialog :open="showDeleteBackupDialog" @close="showDeleteBackupDialog = false" headline="确认删除备份">
      <div>确定要删除备份文件 "{{ deletingBackup?.name }}" 吗？此操作无法撤销。</div>

      <mdui-button slot="action" variant="text" @click="showDeleteBackupDialog = false">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" color="error" @click="confirmDeleteBackup">
        删除
      </mdui-button>
    </mdui-dialog>

    <!-- 添加服务器对话框 -->
    <mdui-dialog :open="showAddDialog" @close="showAddDialog = false" headline="添加服务器">
      <div class="dialog-content">
        <mdui-text-field label="服务器名称" :value="newServer.name" @input="newServer.name = $event.target.value"
          required></mdui-text-field>

        <mdui-text-field label="服务器地址" :value="newServer.address" @input="newServer.address = $event.target.value"
          required></mdui-text-field>

        <mdui-text-field label="端口" type="number" :value="newServer.port"
          @input="newServer.port = parseInt($event.target.value) || 25565"></mdui-text-field>

        <mdui-select label="服务器类型" :value="newServer.type" @change="newServer.type = $event.target.value">
          <mdui-menu-item value="auto">自动检测</mdui-menu-item>
          <mdui-menu-item value="java">Java版</mdui-menu-item>
          <mdui-menu-item value="bedrock">基岩版</mdui-menu-item>
        </mdui-select>

        <mdui-text-field label="描述" :value="newServer.description" @input="newServer.description = $event.target.value"
          rows="3"></mdui-text-field>
      </div>

      <mdui-button slot="action" variant="text" @click="showAddDialog = false">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" @click="addServer">
        添加
      </mdui-button>
    </mdui-dialog>

    <!-- 编辑服务器对话框 -->
    <mdui-dialog :open="showEditDialog" @close="showEditDialog = false" headline="编辑服务器">
      <div class="dialog-content">
        <mdui-text-field label="服务器名称" :value="editingServer.name" @input="editingServer.name = $event.target.value"
          required></mdui-text-field>

        <mdui-text-field label="描述" :value="editingServer.description"
          @input="editingServer.description = $event.target.value" rows="3"></mdui-text-field>
      </div>

      <mdui-button slot="action" variant="text" @click="showEditDialog = false">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" @click="updateServer">
        保存
      </mdui-button>
    </mdui-dialog>

    <!-- 删除确认对话框 -->
    <mdui-dialog :open="showDeleteDialog" @close="showDeleteDialog = false" headline="确认删除">
      <div>确定要删除服务器 "{{ deletingServer?.name }}" 吗？此操作无法撤销。</div>

      <mdui-button slot="action" variant="text" @click="showDeleteDialog = false">
        取消
      </mdui-button>
      <mdui-button slot="action" variant="filled" color="error" @click="confirmDelete">
        删除
      </mdui-button>
    </mdui-dialog>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useServerStore } from '../stores/server'
import { useAuthStore, api } from '../stores/auth'
import { snackbar } from 'mdui'

export default {
  name: 'Admin',
  setup() {
    const serverStore = useServerStore()
    const authStore = useAuthStore()

    // 服务器相关状态
    const servers = ref([])
    const searchText = ref('')
    const showAddDialog = ref(false)
    const showEditDialog = ref(false)
    const showDeleteDialog = ref(false)

    const newServer = ref({
      name: '',
      address: '',
      port: 25565,
      type: 'auto',
      description: ''
    })

    const editingServer = ref({})
    const deletingServer = ref(null)

    // 数据库管理相关状态
    const databaseStats = ref(null)
    const isOptimizing = ref(false)
    const isBackingUp = ref(false)
    const isLoadingBackups = ref(false)
    const isCleaningUp = ref(false)
    const backups = ref([])
    const showBackupsDialog = ref(false)
    const showRestoreDialog = ref(false)
    const showDeleteBackupDialog = ref(false)
    const restoringBackup = ref(null)
    const deletingBackup = ref(null)

    const filteredServers = computed(() => {
      if (!searchText.value) return servers.value
      return servers.value.filter(server =>
        server.name.toLowerCase().includes(searchText.value.toLowerCase()) ||
        server.address.toLowerCase().includes(searchText.value.toLowerCase())
      )
    })

    // 服务器管理方法
    const loadServers = async () => {
      try {
        const response = await serverStore.getServers()
        servers.value = response.data
      } catch (error) {
        console.error('加载服务器列表失败:', error)
        snackbar({ message: '加载服务器列表失败', closeable: true })
      }
    }

    const addServer = async () => {
      try {
        await serverStore.createServer(newServer.value)
        showAddDialog.value = false
        newServer.value = {
          name: '',
          address: '',
          port: 25565,
          type: 'auto',
          description: ''
        }
        await loadServers()
        snackbar({ message: '服务器添加成功', closeable: true })
      } catch (error) {
        console.error('添加服务器失败:', error)
        snackbar({ message: '添加服务器失败', closeable: true })
      }
    }

    const editServer = (server) => {
      editingServer.value = { ...server }
      showEditDialog.value = true
    }

    const updateServer = async () => {
      try {
        await serverStore.updateServer(editingServer.value.id, editingServer.value)
        showEditDialog.value = false
        await loadServers()
        snackbar({ message: '服务器更新成功', closeable: true })
      } catch (error) {
        console.error('更新服务器失败:', error)
        snackbar({ message: '更新服务器失败', closeable: true })
      }
    }

    const deleteServer = (server) => {
      deletingServer.value = server
      showDeleteDialog.value = true
    }

    const confirmDelete = async () => {
      try {
        await serverStore.deleteServer(deletingServer.value.id)
        showDeleteDialog.value = false
        deletingServer.value = null
        await loadServers()
        snackbar({ message: '服务器删除成功', closeable: true })
      } catch (error) {
        console.error('删除服务器失败:', error)
        snackbar({ message: '删除服务器失败', closeable: true })
      }
    }

    const pingServer = async (server) => {
      try {
        await serverStore.pingServer(server.id)
        await loadServers()
        snackbar({ message: '服务器Ping成功', closeable: true })
      } catch (error) {
        console.error('Ping服务器失败:', error)
        snackbar({ message: 'Ping服务器失败', closeable: true })
      }
    }

    // 数据库管理方法
    const loadDatabaseStats = async () => {
      try {
        const response = await api.get('/database/stats')
        if (response.data.success) {
          databaseStats.value = response.data.data
        }
      } catch (error) {
        console.error('加载数据库统计失败:', error)
      }
    }

    const optimizeDatabase = async () => {
      try {
        isOptimizing.value = true
        const response = await api.post('/database/optimize')
        if (response.data.success) {
          snackbar({
            message: `数据库优化完成：删除了 ${response.data.data.deleted_records} 条记录，节省空间 ${formatFileSize(response.data.data.space_saved)}`,
            closeable: true
          })
          await loadDatabaseStats()
        }
      } catch (error) {
        console.error('数据库优化失败:', error)
        snackbar({ message: '数据库优化失败', closeable: true })
      } finally {
        isOptimizing.value = false
      }
    }

    const createBackup = async () => {
      try {
        isBackingUp.value = true
        const response = await api.post('/backup/create')
        if (response.data.success) {
          snackbar({
            message: `备份创建成功：${response.data.data.backup_path}`,
            closeable: true
          })
          // 总是刷新备份列表，无论对话框是否打开
          if (showBackupsDialog.value) {
            await loadBackups()
          }
        }
      } catch (error) {
        console.error('创建备份失败:', error)
        snackbar({ message: '创建备份失败', closeable: true })
      } finally {
        isBackingUp.value = false
      }
    }

    const openBackupsDialog = async () => {
      showBackupsDialog.value = true
      await loadBackups()
    }

    const loadBackups = async () => {
      try {
        isLoadingBackups.value = true
        const response = await api.get('/backup/list')
        if (response.data.success) {
          // 按创建时间从新到旧排序
          backups.value = (response.data.data.backups || []).sort((a, b) => {
            return new Date(b.created_time) - new Date(a.created_time)
          })
        }
      } catch (error) {
        console.error('加载备份列表失败:', error)
        snackbar({ message: '加载备份列表失败', closeable: true })
      } finally {
        isLoadingBackups.value = false
      }
    }

    const cleanupBackups = async () => {
      try {
        isCleaningUp.value = true
        const response = await api.post('/backup/cleanup', {
          keep_days: 30
        })
        if (response.data.success) {
          const data = response.data.data
          if (data.deleted_count > 0) {
            snackbar({
              message: `清理完成：删除了 ${data.deleted_count} 个备份文件，释放空间 ${formatFileSize(data.space_freed)}`,
              closeable: true
            })
          } else {
            snackbar({
              message: '清理完成：没有需要清理的旧备份文件',
              closeable: true
            })
          }
          await loadBackups()
        }
      } catch (error) {
        console.error('清理备份失败:', error)
        snackbar({ message: '清理备份失败', closeable: true })
      } finally {
        isCleaningUp.value = false
      }
    }

    const restoreBackup = (backup) => {
      restoringBackup.value = backup
      showRestoreDialog.value = true
    }

    const confirmRestore = async () => {
      try {
        const response = await api.post('/backup/restore', {
          backup_path: restoringBackup.value.path
        })
        if (response.data.success) {
          snackbar({
            message: `备份恢复成功！原数据库已备份至：${response.data.data.current_db_backup_path}`,
            closeable: true
          })
          // 显示重启建议
          setTimeout(() => {
            snackbar({
              message: '重要提示：建议重启应用程序以确保数据库连接正常工作',
              closeable: true,
              timeout: 8000
            })
          }, 2000)
          showRestoreDialog.value = false
          restoringBackup.value = null
          await loadDatabaseStats()
        }
      } catch (error) {
        console.error('恢复备份失败:', error)
        snackbar({ message: '恢复备份失败', closeable: true })
      }
    }

    const deleteBackup = (backup) => {
      deletingBackup.value = backup
      showDeleteBackupDialog.value = true
    }

    const confirmDeleteBackup = async () => {
      try {
        const response = await api.delete('/backup/delete', {
          data: { backup_path: deletingBackup.value.path }
        })
        if (response.data.success) {
          snackbar({ message: '备份删除成功', closeable: true })
          showDeleteBackupDialog.value = false
          deletingBackup.value = null
          await loadBackups()
        }
      } catch (error) {
        console.error('删除备份失败:', error)
        snackbar({ message: '删除备份失败', closeable: true })
      }
    }

    // 工具方法
    const formatFileSize = (bytes) => {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString('zh-CN')
    }

    onMounted(() => {
      loadServers()
      loadDatabaseStats()
    })

    return {
      // 服务器管理
      servers,
      searchText,
      filteredServers,
      showAddDialog,
      showEditDialog,
      showDeleteDialog,
      newServer,
      editingServer,
      deletingServer,
      addServer,
      editServer,
      updateServer,
      deleteServer,
      confirmDelete,
      pingServer,

      // 数据库管理
      databaseStats,
      isOptimizing,
      isBackingUp,
      isLoadingBackups,
      isCleaningUp,
      backups,
      showBackupsDialog,
      showRestoreDialog,
      showDeleteBackupDialog,
      restoringBackup,
      deletingBackup,
      optimizeDatabase,
      createBackup,
      openBackupsDialog,
      loadBackups,
      cleanupBackups,
      restoreBackup,
      confirmRestore,
      deleteBackup,
      confirmDeleteBackup,
      formatFileSize,
      formatDate
    }
  }
}
</script>

<style scoped>
.admin {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: clamp(16px, 2vw, 20px);
  box-sizing: border-box;
}

.admin-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: clamp(16px, 3vw, 24px);
  gap: 16px;
  flex-wrap: wrap;
}

.admin-header h1 {
  margin: 0;
  font-size: clamp(24px, 4vw, 32px);
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
  flex: 1;
  min-width: 200px;
}

.servers-table {
  margin-bottom: 24px;
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: clamp(16px, 2vw, 24px);
  border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
  gap: 16px;
  flex-wrap: wrap;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
}

.dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0 24px;
  min-width: 300px;
}

/* 新的表格样式 */
.servers-table-container {
  background-color: rgb(var(--mdui-color-surface));
  border-radius: 12px;
  overflow: hidden;
}

.table-header {
  background-color: rgb(var(--mdui-color-surface-variant));
  border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
}

.header-row {
  display: grid;
  grid-template-columns: 2fr 2fr 1fr 1fr 1fr 100px;
  gap: clamp(8px, 1.5vw, 16px);
  padding: clamp(12px, 2vw, 16px) clamp(16px, 2vw, 24px);
}

.header-cell {
  font-weight: 600;
  font-size: clamp(12px, 1.2vw, 14px);
  color: rgb(var(--mdui-color-on-surface-variant));
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.table-body {
  background-color: rgb(var(--mdui-color-surface));
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 2fr 1fr 1fr 1fr 100px;
  gap: clamp(8px, 1.5vw, 16px);
  padding: clamp(12px, 2vw, 16px) clamp(16px, 2vw, 24px);
  border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.table-row:hover {
  background-color: rgb(var(--mdui-color-surface-container-high));
}

.table-row:last-child {
  border-bottom: none;
}

.table-cell {
  display: flex;
  align-items: center;
}

.server-name {
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
}

.server-address {
  font-family: monospace;
  color: rgb(var(--mdui-color-on-surface-variant));
  font-size: 14px;
}

.player-count {
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
}

.action-buttons {
  display: flex;
  gap: 4px;
}

.empty-state {
  padding: 64px 24px;
  text-align: center;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.empty-icon {
  font-size: 64px;
  opacity: 0.5;
  margin-bottom: 16px;
}

.empty-state p {
  margin: 0 0 24px 0;
  font-size: 16px;
}

@media (max-width: 1024px) {

  .header-row,
  .table-row {
    grid-template-columns: 2fr 2fr 1fr 1fr 100px;
    font-size: 14px;
  }

  .players-col {
    display: none;
  }
}

@media (max-width: 768px) {

  .header-row,
  .table-row {
    grid-template-columns: 1fr;
    gap: 8px;
    padding: 12px 16px;
  }

  .header-cell {
    display: none;
  }

  .table-cell {
    padding: 8px 0;
    min-height: 36px;
    border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
    position: relative;
  }

  .table-cell:last-child {
    border-bottom: none;
  }

  .table-cell::before {
    content: attr(data-label);
    font-weight: 600;
    color: rgb(var(--mdui-color-on-surface-variant));
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
  }

  .card-header :deep(mdui-text-field) {
    width: 100%;
  }

  .action-buttons {
    position: absolute;
    right: 0;
    top: 50%;
    transform: translateY(-50%);
  }

  .server-name {
    padding-right: 100px;
  }
}

@media (max-width: 480px) {
  .dialog-content {
    min-width: unset;
    width: 100%;
  }

  .action-buttons {
    gap: 0;
  }

  :deep(.mdui-button-icon) {
    --mdui-comp-button-icon-size: 36px;
  }
}

/* 数据库管理样式 */
.database-management {
  margin-bottom: 24px;
  width: 100%;
}

.database-stats {
  padding: 24px;
  border-bottom: 1px solid rgb(var(--mdui-color-outline-variant));
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 24px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background-color: rgb(var(--mdui-color-surface-container-low));
  border-radius: 12px;
}

.stat-icon {
  font-size: 32px;
  color: rgb(var(--mdui-color-primary));
}

.stat-info {
  flex: 1;
}

.stat-number {
  font-size: 24px;
  font-weight: 700;
  color: rgb(var(--mdui-color-on-surface));
  margin-bottom: 4px;
}

.stat-label {
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.database-actions {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.action-group {
  padding: 24px;
  background-color: rgb(var(--mdui-color-surface-container-low));
  border-radius: 12px;
}

.action-group h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
}

.action-group p {
  margin: 0 0 16px 0;
  color: rgb(var(--mdui-color-on-surface-variant));
  line-height: 1.5;
}

.backup-controls {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.backup-content {
  padding: 0 24px;
  max-height: 400px;
  overflow-y: auto;
}

.backup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.backup-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.backup-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background-color: rgb(var(--mdui-color-surface-container-low));
  border-radius: 8px;
  border: 1px solid rgb(var(--mdui-color-outline-variant));
}

.backup-info {
  flex: 1;
}

.backup-name {
  font-weight: 500;
  color: rgb(var(--mdui-color-on-surface));
  margin-bottom: 4px;
}

.backup-details {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: rgb(var(--mdui-color-on-surface-variant));
  flex-wrap: wrap;
}

.backup-actions {
  display: flex;
  gap: 4px;
}

.empty-backups {
  text-align: center;
  padding: 48px 24px;
  color: rgb(var(--mdui-color-on-surface-variant));
}

.empty-backups .empty-icon {
  font-size: 48px;
  opacity: 0.5;
  margin-bottom: 16px;
}

.empty-backups p {
  margin: 0;
  font-size: 16px;
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 16px;
  }

  .stat-item {
    padding: 12px;
    gap: 12px;
  }

  .stat-icon {
    font-size: 28px;
  }

  .stat-number {
    font-size: 20px;
  }

  .database-actions {
    gap: 24px;
  }

  .action-group {
    padding: 16px;
  }

  .backup-controls {
    flex-direction: column;
  }

  .backup-controls mdui-button {
    width: 100%;
  }

  .backup-header {
    flex-direction: column;
    align-items: stretch;
  }

  .backup-header mdui-button {
    width: 100%;
  }

  .backup-item {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .backup-actions {
    align-self: flex-end;
  }

  .backup-details {
    flex-direction: column;
    gap: 4px;
  }
}
</style>