<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="12">
        <div class="card-box">
          <h4 class="block-title">修改密码</h4>
          <el-form label-width="90px" style="max-width:380px">
            <el-form-item label="原密码">
              <el-input v-model="pw.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="pw.new_password" type="password" show-password placeholder="至少 6 位" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePw">保存</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-col>
      <el-col :span="12">
        <div class="card-box">
          <h4 class="block-title">App 接入信息</h4>
          <el-alert type="info" :closable="false" style="margin-bottom:14px"
            title="App 端调用验证 API 时使用以下凭据做 HMAC-SHA256 签名，请妥善保管 app_secret，不要写入客户端代码。" />
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="app_id"><span class="mono">{{ appkey.app_id }}</span></el-descriptions-item>
            <el-descriptions-item label="app_secret">
              <span class="mono">{{ showSecret ? appkey.app_secret : '••••••••••••••••' }}</span>
              <el-button link type="primary" size="small" style="margin-left:8px" @click="showSecret = !showSecret">
                {{ showSecret ? '隐藏' : '显示' }}
              </el-button>
            </el-descriptions-item>
            <el-descriptions-item label="离线宽限期">{{ appkey.grace_hours }} 小时</el-descriptions-item>
          </el-descriptions>
          <h4 class="block-title" style="margin-top:20px">签名算法</h4>
          <div class="sign-doc mono">
            X-Signature = HMAC-SHA256(app_secret, timestamp + nonce + body).hex()
          </div>
          <p style="color:#909399; font-size:12px; margin-top:8px; line-height:1.7">
            请求头：X-App-Id / X-Timestamp（毫秒，偏差 ≤ 5 分钟）/ X-Nonce（随机串，5 分钟窗口防重放）/ X-Signature。<br>
            接口：POST /api/v1/activate · /api/v1/verify · /api/v1/renew
          </p>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'

const pw = reactive({ old_password: '', new_password: '' })
const appkey = ref({})
const showSecret = ref(false)

onMounted(async () => {
  appkey.value = await http.get('/appkey')
})

async function changePw() {
  if (!pw.old_password || !pw.new_password) return ElMessage.warning('请填写完整')
  if (pw.new_password.length < 6) return ElMessage.warning('新密码至少 6 位')
  await http.post('/password', pw)
  ElMessage.success('密码已修改，下次登录请使用新密码')
  pw.old_password = ''
  pw.new_password = ''
}
</script>

<style scoped>
.block-title { font-size: 14px; color: #303133; margin-bottom: 14px; }
.sign-doc {
  background: #f8f9fb; border: 1px solid #ebeef5; border-radius: 4px;
  padding: 10px 12px; font-size: 12px; color: #e63946; word-break: break-all;
}
</style>
