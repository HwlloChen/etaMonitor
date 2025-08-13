<template>
  <div class="admin">
    <div class="admin-header">
      <h1>管理面板</h1>
      <mdui-button 
        variant="filled" 
        icon="add" 
        @click="showAddDialog = true"
      >
        添加服务器
      </mdui-button>
    </div>

    <!-- 服务器管理 -->
    <mdui-card class="servers-table">
      <div class="card-header">
        <h2>服务器管理</h2>
        <mdui-text-field 
          placeholder="搜索服务器..."
          icon="search"
          :value="searchText"
          @input="searchText = $event.target.value"
        ></mdui-text-field>
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
          
          <div 
            v-for="server in filteredServers" 
            :key="server.id"
            class="table-row"
            @click="$router.push(`/server/${server.id}`)"
          >
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
              <mdui-chip 
                :color="server.status === 'online' ? 'primary' : 'error'"
                variant="filled"
              >
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
                <mdui-button-icon 
                  icon="edit" 
                  @click="editServer(server)"
                  variant="standard"
                ></mdui-button-icon>
                <mdui-button-icon 
                  icon="refresh" 
                  @click="pingServer(server)"
                  variant="standard"
                ></mdui-button-icon>
                <mdui-button-icon 
                  icon="delete" 
                  @click="deleteServer(server)"
                  variant="standard"
                ></mdui-button-icon>
              </div>
            </div>
          </div>
        </div>
      </div>
    </mdui-card>

    <!-- 添加服务器对话框 -->
    <mdui-dialog :open="showAddDialog" @close="showAddDialog = false" headline="添加服务器">
      <div class="dialog-content">
        <mdui-text-field 
          label="服务器名称" 
          :value="newServer.name"
          @input="newServer.name = $event.target.value"
          required
        ></mdui-text-field>
        
        <mdui-text-field 
          label="服务器地址" 
          :value="newServer.address"
          @input="newServer.address = $event.target.value"
          required
        ></mdui-text-field>
        
        <mdui-text-field 
          label="端口" 
          type="number" 
          :value="newServer.port"
          @input="newServer.port = parseInt($event.target.value) || 25565"
        ></mdui-text-field>
        
        <mdui-select label="服务器类型" :value="newServer.type" @change="newServer.type = $event.target.value">
          <mdui-menu-item value="auto">自动检测</mdui-menu-item>
          <mdui-menu-item value="java">Java版</mdui-menu-item>
          <mdui-menu-item value="bedrock">基岩版</mdui-menu-item>
        </mdui-select>
        
        <mdui-text-field 
          label="描述" 
          :value="newServer.description"
          @input="newServer.description = $event.target.value"
          rows="3"
        ></mdui-text-field>
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
        <mdui-text-field 
          label="服务器名称" 
          :value="editingServer.name"
          @input="editingServer.name = $event.target.value"
          required
        ></mdui-text-field>
        
        <mdui-text-field 
          label="描述" 
          :value="editingServer.description"
          @input="editingServer.description = $event.target.value"
          rows="3"
        ></mdui-text-field>
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

export default {
  name: 'Admin',
  setup() {
    const serverStore = useServerStore()
    
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
    
    const filteredServers = computed(() => {
      if (!searchText.value) return servers.value
      return servers.value.filter(server => 
        server.name.toLowerCase().includes(searchText.value.toLowerCase()) ||
        server.address.toLowerCase().includes(searchText.value.toLowerCase())
      )
    })
    
    const loadServers = async () => {
      try {
        const response = await serverStore.getServers()
        servers.value = response.data
      } catch (error) {
        console.error('加载服务器列表失败:', error)
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
      } catch (error) {
        console.error('添加服务器失败:', error)
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
      } catch (error) {
        console.error('更新服务器失败:', error)
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
      } catch (error) {
        console.error('删除服务器失败:', error)
      }
    }
    
    const pingServer = async (server) => {
      try {
        await serverStore.pingServer(server.id)
        await loadServers()
      } catch (error) {
        console.error('Ping服务器失败:', error)
      }
    }
    
    onMounted(() => {
      loadServers()
    })
    
    return {
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
      pingServer
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
</style>