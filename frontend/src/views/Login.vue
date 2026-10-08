<template>
  <div class="login-wrap">
    <div class="login-card">
      <h2>发卡系统</h2>
      <p class="sub">卡密管理后台 · Card Admin</p>
      <el-form :model="form" @keyup.enter="submit">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" size="large" autofocus>
            <template #prefix><el-icon><User /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" size="large" show-password>
            <template #prefix><el-icon><Lock /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-button type="primary" size="large" style="width:100%" :loading="loading" @click="submit">登 录</el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import axios from 'axios'

const router = useRouter()
const form = reactive({ username: '', password: '' })
const loading = ref(false)

async function submit() {
  if (!form.username || !form.password) return ElMessage.warning('请输入用户名与密码')
  loading.value = true
  try {
    const res = await axios.post('/api/admin/login', form)
    localStorage.setItem('token', res.data.data.token)
    localStorage.setItem('username', form.username)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  height: 100%; display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg, #1d3557 0%, #457b9d 100%);
}
.login-card {
  width: 380px; background: #fff; border-radius: 12px; padding: 40px 36px 32px;
  box-shadow: 0 12px 40px rgba(0,0,0,.25);
}
.login-card h2 { text-align: center; margin-bottom: 4px; color: #1d3557; }
.login-card .sub { text-align: center; color: #909399; font-size: 13px; margin-bottom: 28px; }
</style>
