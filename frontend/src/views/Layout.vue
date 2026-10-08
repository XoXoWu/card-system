<template>
  <el-container style="height:100%">
    <el-aside width="220px" class="aside">
      <div class="logo">
        <span class="logo-mark">C</span>
        <span>发卡系统</span>
      </div>
      <el-menu :default-active="$route.path" router background-color="#1d3557" text-color="#cdd7e2" active-text-color="#fff">
        <el-menu-item index="/dashboard"><el-icon><DataLine /></el-icon>数据看板</el-menu-item>
        <el-menu-item index="/generate"><el-icon><MagicStick /></el-icon>卡密生成</el-menu-item>
        <el-menu-item index="/cards"><el-icon><Tickets /></el-icon>卡密列表</el-menu-item>
        <el-menu-item index="/plans"><el-icon><PriceTag /></el-icon>套餐管理</el-menu-item>
        <el-menu-item index="/oplogs"><el-icon><Document /></el-icon>操作日志</el-menu-item>
        <el-menu-item index="/settings"><el-icon><Setting /></el-icon>系统设置</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span class="crumb">{{ $route.meta.title || '' }}</span>
        <el-dropdown @command="onCommand">
          <span class="user">
            <el-icon><UserFilled /></el-icon>{{ username }}
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main style="padding:0; overflow:auto">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { DataLine, MagicStick, Tickets, PriceTag, Document, Setting, UserFilled, ArrowDown } from '@element-plus/icons-vue'

const router = useRouter()
const username = computed(() => localStorage.getItem('username') || 'admin')

function onCommand(cmd) {
  if (cmd === 'logout') {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    ElMessage.success('已退出')
    router.push('/login')
  }
}
</script>

<style scoped>
.aside { background: #1d3557; }
.logo {
  height: 60px; display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 17px; font-weight: 600; gap: 8px;
}
.logo-mark {
  width: 28px; height: 28px; border-radius: 6px; background: #e63946;
  display: inline-flex; align-items: center; justify-content: center;
  font-weight: 800; font-size: 15px;
}
.aside :deep(.el-menu) { border-right: none; }
.aside :deep(.el-menu-item.is-active) { background: #e63946; }
.header {
  background: #fff; display: flex; align-items: center; justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0,21,41,.08); z-index: 5;
}
.crumb { font-size: 15px; font-weight: 600; color: #303133; }
.user { display: inline-flex; align-items: center; gap: 6px; cursor: pointer; color: #606266; font-size: 14px; }
</style>
