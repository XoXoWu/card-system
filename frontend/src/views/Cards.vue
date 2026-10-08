<template>
  <div class="page">
    <div class="card-box">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="卡号末4位 / 批次 / 备注 / 完整卡密" clearable @keyup.enter="load" style="width:230px" />
        <el-select v-model="query.status" placeholder="状态" @change="load">
          <el-option label="全部状态" value="all" />
          <el-option label="未激活" value="unused" />
          <el-option label="使用中" value="active" />
          <el-option label="已过期" value="expired" />
          <el-option label="已封禁" value="banned" />
          <el-option label="已作废" value="revoked" />
        </el-select>
        <el-select v-model="query.type" placeholder="类型" @change="load">
          <el-option label="全部类型" value="" />
          <el-option label="普通卡" value="normal" />
          <el-option label="测试卡" value="test" />
        </el-select>
        <el-select v-model="query.plan_id" placeholder="套餐" @change="load">
          <el-option label="全部套餐" value="all" />
          <el-option v-for="p in plans" :key="p.id" :label="p.name" :value="String(p.id)" />
        </el-select>
        <el-button type="primary" @click="load">查询</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="卡号" width="130">
          <template #default="{ row }">
            <span class="mono">****{{ row.key_tail }}</span>
            <el-tag v-if="row.is_test" type="warning" size="small" style="margin-left:4px">测试</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="plan_name" label="套餐" width="120" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="设备" width="70">
          <template #default="{ row }">
            <span :class="{ 'device-full': row.device_count >= row.max_devices }">{{ row.device_count }}/{{ row.max_devices }}</span>
          </template>
        </el-table-column>
        <el-table-column label="激活时间" width="105">
          <template #default="{ row }">{{ row.activated_at ? fmtDate(row.activated_at) : '—' }}</template>
        </el-table-column>
        <el-table-column label="到期时间" width="105">
          <template #default="{ row }">{{ row.expire_at ? fmtDate(row.expire_at) : '—' }}</template>
        </el-table-column>
        <el-table-column prop="batch_no" label="批次号" width="150" show-overflow-tooltip />
        <el-table-column prop="note" label="备注" min-width="110" show-overflow-tooltip>
          <template #default="{ row }">{{ row.note || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openDetail(row)">详情</el-button>
            <el-button v-if="row.status !== 'banned' && row.status !== 'revoked'" link type="danger" size="small" @click="ban(row)">封禁</el-button>
            <el-button v-if="row.status === 'banned'" link type="success" size="small" @click="unban(row)">解封</el-button>
            <el-button v-if="row.status !== 'revoked'" link type="warning" size="small" @click="revoke(row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px; justify-content:flex-end" layout="total, prev, pager, next, sizes"
        :total="total" v-model:current-page="query.page" v-model:page-size="query.size"
        :page-sizes="[20, 50, 100]" @change="load" />
    </div>

    <!-- 详情抽屉 -->
    <el-drawer v-model="detailVisible" title="卡密详情" size="520px">
      <template v-if="detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="卡号" :span="2">
            <span class="mono">****{{ detail.card.key_tail }}</span>
            <el-tag v-if="detail.card.is_test" type="warning" size="small" style="margin-left:6px">测试卡</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="套餐">{{ detail.card.plan_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusType(detail.card.status)" size="small">{{ statusText(detail.card.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="时长">{{ detail.card.is_test ? '激活后 24 小时' : detail.card.duration_days + ' 天' }}</el-descriptions-item>
          <el-descriptions-item label="设备上限">{{ detail.card.max_devices }} 台</el-descriptions-item>
          <el-descriptions-item label="激活时间">{{ detail.card.activated_at ? fmtTime(detail.card.activated_at) : '—' }}</el-descriptions-item>
          <el-descriptions-item label="到期时间">{{ detail.card.expire_at ? fmtTime(detail.card.expire_at) : '—' }}</el-descriptions-item>
          <el-descriptions-item label="批次号" :span="2"><span class="mono">{{ detail.card.batch_no }}</span></el-descriptions-item>
        </el-descriptions>

        <h4 class="block-title" style="margin-top:20px">绑定设备（{{ boundCount }}/{{ detail.card.max_devices }}）</h4>
        <el-table :data="detail.devices" size="small" empty-text="暂无设备绑定">
          <el-table-column prop="device_info" label="设备" min-width="150">
            <template #default="{ row }">{{ row.device_info || row.device_id }}</template>
          </el-table-column>
          <el-table-column label="最近活跃" width="140">
            <template #default="{ row }">{{ fmtTime(row.last_seen) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.unbound_at ? 'info' : 'success'" size="small">{{ row.unbound_at ? '已解绑' : '绑定中' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="" width="70">
            <template #default="{ row }">
              <el-button v-if="!row.unbound_at" link type="danger" size="small" @click="unbind(row)">解绑</el-button>
            </template>
          </el-table-column>
        </el-table>

        <h4 class="block-title" style="margin-top:20px">备注</h4>
        <el-input v-model="detail.card.note" type="textarea" :rows="2" maxlength="100" placeholder="渠道、买家信息等" />
        <el-button type="primary" size="small" style="margin-top:10px" @click="saveNote">保存备注</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const plans = ref([])
const list = ref([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ keyword: '', status: 'all', type: '', plan_id: 'all', page: 1, size: 20 })

const detailVisible = ref(false)
const detail = ref(null)
const boundCount = computed(() => (detail.value?.devices || []).filter(d => !d.unbound_at).length)

onMounted(() => { load(); http.get('/plans').then(p => plans.value = p) })

async function load() {
  loading.value = true
  try {
    const params = Object.fromEntries(Object.entries(query).filter(([, v]) => v !== '' && v != null))
    const data = await http.get('/cards', { params })
    list.value = data.list || []
    total.value = data.total
  } finally {
    loading.value = false
  }
}

async function openDetail(row) {
  detail.value = await http.get(`/cards/${row.id}`)
  detailVisible.value = true
}

async function ban(row) {
  await ElMessageBox.confirm(`确认封禁 ****${row.key_tail}？封禁后 App 端将立即无法通过验证。`, '封禁确认', { type: 'warning' })
  await http.post(`/cards/${row.id}/ban`)
  ElMessage.success('已封禁')
  load()
}

async function unban(row) {
  await http.post(`/cards/${row.id}/unban`)
  ElMessage.success('已解封')
  load()
}

async function revoke(row) {
  await ElMessageBox.confirm(`确认作废 ****${row.key_tail}？作废不可恢复。`, '作废确认', { type: 'warning' })
  await http.post(`/cards/${row.id}/revoke`)
  ElMessage.success('已作废')
  load()
}

async function unbind(device) {
  await ElMessageBox.confirm(`确认解绑设备「${device.device_info || device.device_id}」？解绑后该设备将无法通过验证。`, '解绑确认', { type: 'warning' })
  await http.post(`/cards/${detail.value.card.id}/devices/${device.id}/unbind`)
  ElMessage.success('已解绑')
  detail.value = await http.get(`/cards/${detail.value.card.id}`)
}

async function saveNote() {
  await http.post(`/cards/${detail.value.card.id}/note`, { note: detail.value.card.note })
  ElMessage.success('备注已保存')
  load()
}

function statusType(s) {
  return { unused: 'primary', active: 'success', expired: 'info', banned: 'danger', revoked: 'warning' }[s] || 'info'
}
function statusText(s) {
  return { unused: '未激活', active: '使用中', expired: '已过期', banned: '已封禁', revoked: '已作废' }[s] || s
}
function fmtDate(ts) {
  return new Date(ts * 1000).toLocaleDateString('zh-CN')
}
function fmtTime(ts) {
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}
</script>

<style scoped>
.device-full { color: #e63946; font-weight: 600; }
.block-title { font-size: 13px; color: #606266; margin-bottom: 10px; }
</style>
