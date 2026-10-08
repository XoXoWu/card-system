<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="card-box">
          <h4 class="block-title">批量生成卡密</h4>
          <el-form :model="form" label-width="90px">
            <el-form-item label="选择套餐">
              <el-select v-model="form.plan_id" style="width:100%">
                <el-option v-for="p in plans" :key="p.id" :value="p.id"
                  :label="`${p.name} · ${p.days} 天 · ${p.max_devices} 设备`" :disabled="!p.enabled" />
              </el-select>
            </el-form-item>
            <el-form-item label="卡密前缀">
              <el-input v-model="form.prefix" placeholder="选填，0–8 位大写字母数字，如 VIP" maxlength="8" style="width:260px"
                @input="form.prefix = form.prefix.toUpperCase()" />
              <div class="preview mono">{{ form.prefix ? form.prefix + '-XXXX-XXXX-XXXX' : 'XXXX-XXXX-XXXX' }}</div>
            </el-form-item>
            <el-form-item label="生成数量">
              <el-input-number v-model="form.count" :min="1" :max="5000" />
              <span class="hint">单批上限 5000 张，超过建议分批</span>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.note" placeholder="如：淘宝渠道 9 月批次" maxlength="50" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="loading" @click="generate">生成卡密</el-button>
              <el-button type="warning" plain @click="testVisible = true">发放测试卡</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-col>

      <el-col :span="10">
        <div class="card-box">
          <h4 class="block-title">生成结果</h4>
          <el-empty v-if="!result" description="尚未生成，卡密明文仅在生成当次展示" />
          <template v-else>
            <div class="result-meta">
              <span>批次号 <b class="mono">{{ result.batch_no }}</b></span>
              <span>共 <b>{{ result.count }}</b> 张</span>
            </div>
            <div class="key-result mono">
              <code v-for="k in result.keys" :key="k">{{ k }}</code>
            </div>
            <div style="margin-top:12px; display:flex; gap:8px">
              <el-button size="small" @click="copy(result.keys)">复制全部</el-button>
              <el-button size="small" @click="exportTxt(result)">导出 TXT</el-button>
              <el-button size="small" @click="exportCsv(result)">导出 CSV</el-button>
            </div>
          </template>
        </div>
      </el-col>
    </el-row>

    <!-- 测试卡弹窗 -->
    <el-dialog v-model="testVisible" title="发放测试卡" width="460px">
      <el-alert type="warning" :closable="false" style="margin-bottom:16px"
        title="测试卡激活后有效期固定 24 小时、限 1 台设备，前缀 TEST，不计入销售统计，不可作为续费卡。" />
      <el-form label-width="90px">
        <el-form-item label="发放数量">
          <el-input-number v-model="testForm.count" :min="1" :max="10" />
        </el-form-item>
        <el-form-item label="用途备注">
          <el-input v-model="testForm.note" placeholder="如：内测 · 给测试同学小王" maxlength="50" />
        </el-form-item>
      </el-form>
      <div v-if="testResult" class="key-result mono" style="margin-bottom:12px">
        <code v-for="k in testResult.keys" :key="k">{{ k }}</code>
      </div>
      <template #footer>
        <el-button @click="testVisible = false">关闭</el-button>
        <el-button type="primary" :loading="testLoading" @click="issueTest">
          {{ testResult ? '再次发放' : '立即发放' }}
        </el-button>
        <el-button v-if="testResult" plain type="primary" @click="copy(testResult.keys)">复制全部</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'

const plans = ref([])
const form = reactive({ plan_id: null, prefix: '', count: 100, note: '' })
const loading = ref(false)
const result = ref(null)

const testVisible = ref(false)
const testForm = reactive({ count: 1, note: '' })
const testLoading = ref(false)
const testResult = ref(null)

onMounted(async () => {
  plans.value = await http.get('/plans')
  if (plans.value.length && !form.plan_id) {
    const first = plans.value.find(p => p.enabled)
    if (first) form.plan_id = first.id
  }
})

async function generate() {
  if (!form.plan_id) return ElMessage.warning('请选择套餐')
  loading.value = true
  try {
    result.value = await http.post('/cards/generate', form)
    ElMessage.success(`已生成 ${result.value.count} 张卡密，请及时复制或导出`)
  } finally {
    loading.value = false
  }
}

async function issueTest() {
  testLoading.value = true
  try {
    testResult.value = await http.post('/cards/test', testForm)
    ElMessage.success(`已发放 ${testResult.value.count} 张测试卡，激活后 24 小时到期`)
  } finally {
    testLoading.value = false
  }
}

function copy(keys) {
  navigator.clipboard.writeText(keys.join('\n')).then(
    () => ElMessage.success(`已复制 ${keys.length} 张卡密`),
    () => ElMessage.error('复制失败，请手动选择复制')
  )
}

function download(content, filename, mime) {
  const blob = new Blob([content], { type: mime })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = filename
  a.click()
  URL.revokeObjectURL(a.href)
}

function exportTxt(r) {
  download(r.keys.join('\r\n'), `${r.batch_no}.txt`, 'text/plain;charset=utf-8')
}

function exportCsv(r) {
  const rows = [['card_key'], ...r.keys.map(k => [k])]
  const csv = '\ufeff' + rows.map(row => row.join(',')).join('\r\n')
  download(csv, `${r.batch_no}.csv`, 'text/csv;charset=utf-8')
}
</script>

<style scoped>
.block-title { margin-bottom: 18px; font-size: 14px; color: #303133; }
.preview { color: #909399; font-size: 12px; margin-top: 6px; }
.hint { color: #909399; font-size: 12px; margin-left: 10px; }
.result-meta { display: flex; justify-content: space-between; color: #606266; font-size: 13px; margin-bottom: 8px; }
</style>
