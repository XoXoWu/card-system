<template>
  <div class="page">
    <div class="card-box">
      <div class="filter-bar">
        <h4 style="flex:1">套餐（销售期限与设备上限）</h4>
        <el-button type="primary" @click="openEdit()">新建套餐</el-button>
      </div>

      <el-table :data="plans" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" width="130">
          <template #default="{ row }">
            {{ row.name }}
            <el-tag v-if="!row.enabled" type="info" size="small">停用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时长" width="100">
          <template #default="{ row }">{{ row.days }} 天</template>
        </el-table-column>
        <el-table-column label="设备上限" width="100">
          <template #default="{ row }">{{ row.max_devices }} 台</template>
        </el-table-column>
        <el-table-column label="参考价" width="110">
          <template #default="{ row }">¥ {{ row.price.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link :type="row.enabled ? 'warning' : 'success'" size="small" @click="toggle(row)">
              {{ row.enabled ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" size="small" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="editVisible" :title="editForm.id ? '编辑套餐' : '新建套餐'" width="460px">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" maxlength="20" placeholder="如：月卡" />
        </el-form-item>
        <el-form-item label="时长（天）">
          <el-input-number v-model="editForm.days" :min="1" :max="3650" />
        </el-form-item>
        <el-form-item label="设备上限">
          <el-input-number v-model="editForm.max_devices" :min="1" :max="10" />
          <span class="hint">激活后可绑定的设备数</span>
        </el-form-item>
        <el-form-item label="参考价（元）">
          <el-input-number v-model="editForm.price" :min="0" :precision="2" :step="1" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editForm.remark" maxlength="50" placeholder="如：90 天 · 双设备" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const plans = ref([])
const loading = ref(false)
const editVisible = ref(false)
const editForm = reactive({ id: null, name: '', days: 30, max_devices: 1, price: 0, remark: '', enabled: true })

onMounted(load)

async function load() {
  loading.value = true
  try {
    plans.value = await http.get('/plans')
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  Object.assign(editForm, row || { id: null, name: '', days: 30, max_devices: 1, price: 0, remark: '', enabled: true })
  editVisible.value = true
}

async function save() {
  if (!editForm.name.trim()) return ElMessage.warning('请输入套餐名称')
  if (editForm.id) {
    await http.put(`/plans/${editForm.id}`, editForm)
  } else {
    await http.post('/plans', editForm)
  }
  ElMessage.success('已保存')
  editVisible.value = false
  load()
}

async function toggle(row) {
  await http.put(`/plans/${row.id}`, { ...row, enabled: !row.enabled })
  ElMessage.success(row.enabled ? '已停用' : '已启用')
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`确认删除套餐「${row.name}」？`, '删除确认', { type: 'warning' })
  await http.delete(`/plans/${row.id}`)
  ElMessage.success('已删除')
  load()
}
</script>

<style scoped>
.hint { color: #909399; font-size: 12px; margin-left: 10px; }
</style>
