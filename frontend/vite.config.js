import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [
    vue({
      template: {
        compilerOptions: {
          // 将所有以 mdui- 开头的标签视为自定义元素
          isCustomElement: (tag) => tag.startsWith('mdui-')
        }
      }
    })
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:11451',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:11451',
        ws: true
      }
    }
  }
})