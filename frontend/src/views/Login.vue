<template>
  <div class="login-container">
    <div class="login-box">
      <h2>🗄️🔮🤖 百宝箱</h2>
      <p class="subtitle">你的随身百宝箱</p>
      <el-form @submit.native.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="username" name="username" placeholder="用户名" prefix-icon="el-icon-user" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" name="password" type="password" placeholder="密码" prefix-icon="el-icon-lock" size="large" @keyup.enter="handleLogin" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" long size="large">登 录</el-button>
        <div class="register-tip">
          还没有账号？<el-button type="text" @click="showRegister = true">立即注册</el-button>
        </div>
      </el-form>
      
      <!-- 注册弹窗 -->
      <el-dialog title="注册新账号" :visible.sync="showRegister" width="400px">
        <el-form ref="registerForm" :model="registerData" label-width="80px">
          <el-form-item label="用户名" required>
            <el-input v-model="registerData.username" name="reg_username" placeholder="请设置用户名" />
          </el-form-item>
          <el-form-item label="密码" required>
            <el-input v-model="registerData.password" name="reg_password" type="password" placeholder="请设置密码（至少6位）" />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="registerData.email" name="reg_email" placeholder="选填" />
          </el-form-item>
        </el-form>
        <span slot="footer">
          <el-button @click="showRegister = false">取消</el-button>
          <el-button type="primary" :loading="registering" @click="handleRegister">注册</el-button>
        </span>
      </el-dialog>
    </div>
  </div>
</template>

<script>
import api from '@/api'

export default {
  data() {
    return {
      username: '',
      password: '',
      loading: false,
      showRegister: false,
      registering: false,
      registerData: {
        username: '',
        password: '',
        email: ''
      }
    }
  },
  methods: {
    async handleLogin() {
      if (!this.username || !this.password) {
        this.$message.warning('请输入用户名和密码')
        return
      }
      this.loading = true
      try {
        const success = await this.$store.dispatch('auth/login', {
          username: this.username,
          password: this.password
        })
        if (success) {
          this.$router.push('/')
        }
      } finally {
        this.loading = false
      }
    },
    async handleRegister() {
      if (!this.registerData.username && !this.registerData.password) {
        this.$message.warning('请填写用户名和密码')
        return
      }
      if (!this.registerData.username) {
        this.$message.warning('请填写用户名')
        return
      }
      if (!this.registerData.password) {
        this.$message.warning('请填写密码')
        return
      }
      if (this.registerData.password.length < 6) {
        this.$message.warning('密码至少6位')
        return
      }
      this.registering = true
      try {
        const res = await api.post(`/weekly-assistant/auth/register`, this.registerData)
        if (res.data.code === 0) {
          this.$message.success('注册成功，请登录')
          this.showRegister = false
          this.username = this.registerData.username
          this.password = ''
          this.registerData = { username: '', password: '', email: '' }
        }
      } catch {
        // 错误已在拦截器提示
      } finally {
        this.registering = false
      }
    }
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-box {
  background: white;
  padding: 48px 40px;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
  width: 380px;
}
.login-box h2 {
  text-align: center;
  margin-bottom: 8px;
  font-size: 24px;
}
.subtitle {
  text-align: center;
  color: #999;
  font-size: 14px;
  margin-bottom: 32px;
}
.register-tip {
  text-align: center;
  margin-top: 16px;
  font-size: 14px;
  color: #666;
}
</style>