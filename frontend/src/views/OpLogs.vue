<template>
  <div class="page">
    <div class="card-box">
      <h4 class="block-title" style="margin-bottom:16px">操作日志</h4>
      <el-table :data="logs" v-loading="loading" stripe>
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作人" width="100">
          <template #default="{ row }">{{ row.admin_name || 'system' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-tag :type="actionType(row.action)" size="small">{{ actionText(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="对象" width="160" show-overflow-tooltip />
        <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ row.detail || '—' }}</template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:16px; justify-content:flex-end" layout="total, prev, pager, next"
        :total="total" v-model:current-page="page" :page-size="20" @change="load" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import http from '../api'

const logs = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await http.get('/oplogs', { params: { page: page.value, size: 20 } })
    logs.value = data.list || []
    total.value = data.total
  } finally {
    loading.value = false
  }
}

const ACTIONS = {
  login: ['登录', ''],
  generate: ['生成卡密', 'primary'],
  test_issue: ['发放测试卡', 'warning'],
  ban: ['封禁', 'danger'],
  unban: ['解封', 'success'],
  revoke: ['作废', 'warning'],
  unbind: ['解绑设备', 'warning'],
  note: ['修改备注', 'info'],
  plan_create: ['新建套餐', 'success'],
  plan_update: ['更新套餐', 'primary'],
  plan_delete: ['删除套餐', 'danger'],
  change_password: ['修改密码', 'info'],
  activate: ['App 激活', 'success'],
  renew: ['App 续费', 'success']
}
function actionText(a) { return ACTIONS[a]?.[0] || a }
function actionType(a) { return ACTIONS[a]?.[1] || 'info' }
function fmtTime(ts) {
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}
</script>

<style scoped>
.block-title { font-size: 14px; color: #303133; }
</style>
