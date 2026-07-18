<template>
  <el-container class="app-container">
    <el-aside :width="isCollapsed ? '64px' : '220px'" class="app-aside">
      <div class="aside-header">
        <span v-show="!isCollapsed" class="aside-title">🗄️🔮🤖 百宝箱</span>
        <el-button
          :icon="isCollapsed ? 'el-icon-s-unfold' : 'el-icon-s-fold'"
          size="mini"
          class="collapse-btn"
          @click="toggleCollapse"
        />
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapsed"
        :collapse-transition="false"
        background-color="#fff"
        text-color="#333"
        active-text-color="#409eff"
        router
      >
        <el-menu-item index="/">
          <i class="el-icon-document-copy"></i>
          <span slot="title">周报AI助手</span>
        </el-menu-item>
        <el-menu-item index="/document-parse">
          <i class="el-icon-reading"></i>
          <span slot="title">文档转换</span>
        </el-menu-item>
        <el-submenu index="/contract">
          <template slot="title">
            <i class="el-icon-document-checked"></i>
            <span slot="title">合同审查</span>
          </template>
          <el-menu-item index="/contract-review">
            <i class="el-icon-upload2"></i>
            <span slot="title">上传审查</span>
          </el-menu-item>
          <el-menu-item index="/contract-history">
            <i class="el-icon-time"></i>
            <span slot="title">我的审查</span>
          </el-menu-item>
        </el-submenu>
      </el-menu>
    </el-aside>
    <el-container class="app-main">
      <div class="top-bar">
        <div class="top-left"></div>
        <div class="top-right">
          <el-dropdown @command="handleUserCommand" trigger="click">
            <span class="el-dropdown-link username">
              {{ currentUser?.username }} <i class="el-icon-arrow-down el-icon--right"></i>
            </span>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item command="profile"><i class="el-icon-setting"></i> 个人中心</el-dropdown-item>
              <el-dropdown-item command="logout" divided><i class="el-icon-switch-button"></i> 退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
      </div>
      <router-view />
    </el-container>

    <!-- 个人中心 -->
    <el-dialog title="个人中心" :visible.sync="profileVisible" width="400px">
      <el-form label-width="80px" size="small">
        <el-form-item label="用户名">
          <el-input :value="currentUser?.username" disabled />
        </el-form-item>
        <el-form-item label="原密码">
          <el-input v-model="pwdForm.oldPassword" placeholder="请输入原密码" :type="pwdShow.old ? 'text' : 'password'">
            <i slot="suffix" class="el-icon-view" style="cursor:pointer" @click="pwdShow.old=!pwdShow.old"></i>
          </el-input>
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.newPassword" placeholder="请输入新密码（至少6位）" :type="pwdShow.new ? 'text' : 'password'">
            <i slot="suffix" class="el-icon-view" style="cursor:pointer" @click="pwdShow.new=!pwdShow.new"></i>
          </el-input>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="profileVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdLoading" @click="handleChangePassword">修改密码</el-button>
      </span>
    </el-dialog>
  </el-container>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'MainLayout',
  data() {
    return {
      isCollapsed: false,
      profileVisible: false,
      pwdLoading: false,
      pwdForm: { oldPassword: '', newPassword: '' },
      pwdShow: { old: false, new: false }
    }
  },
  computed: {
    ...mapGetters('auth', ['currentUser']),
    activeMenu() {
      return this.$route.path
    }
  },
  methods: {
    toggleCollapse() {
      this.isCollapsed = !this.isCollapsed
    },
    handleUserCommand(cmd) {
      if (cmd === 'profile') {
        this.pwdForm = { oldPassword: '', newPassword: '' }
        this.profileVisible = true
      } else if (cmd === 'logout') {
        this.$confirm('确定要退出登录吗？', '退出确认', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          this.$store.dispatch('auth/logout')
          this.$router.push('/login')
        }).catch(() => {})
      }
    },
    async handleChangePassword() {
      if (!this.pwdForm.oldPassword || !this.pwdForm.newPassword) {
        this.$message.warning('请填写完整信息')
        return
      }
      if (this.pwdForm.newPassword.length < 6) {
        this.$message.warning('新密码至少6位')
        return
      }
      this.pwdLoading = true
      try {
        const { default: api } = await import('@/api/index')
        const { SUCCESS_CODE } = await import('@/constants')
        const res = await api.post(`/weekly-assistant/user/change-password`, {
          old_password: this.pwdForm.oldPassword,
          new_password: this.pwdForm.newPassword
        })
        if (res.data.code === SUCCESS_CODE) {
          this.$message.success('密码修改成功')
          this.profileVisible = false
          this.pwdForm = { oldPassword: '', newPassword: '' }
        }
      } catch {} finally {
        this.pwdLoading = false
      }
    }
  }
}
</script>

<style scoped>
.app-container {
  height: 100vh;
}
.app-aside {
  background-color: #fff;
  border-right: 1px solid #e4e7ed;
  transition: width 0.3s;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.aside-header {
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  border-bottom: 1px solid #e4e7ed;
  gap: 4px;
}
.aside-title {
  color: #333;
  font-size: 18px;
  white-space: nowrap;
  overflow: hidden;
}
.collapse-btn {
  background: transparent !important;
  border: none !important;
  color: #666 !important;
  font-size: 18px;
  padding: 0;
}
.collapse-btn:hover {
  color: #409eff !important;
}
.el-menu {
  border-right: none;
  flex: 1;
}
.app-main {
  display: flex;
  flex-direction: column;
  overflow: auto;
}
.top-bar {
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}
.username {
  cursor: pointer;
  color: #333;
  font-size: 14px;
}
.username:hover {
  color: #409eff;
}
</style>
