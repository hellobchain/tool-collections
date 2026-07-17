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
          <span slot="title">文档解析</span>
        </el-menu-item>
        <el-menu-item index="/contract-review">
          <i class="el-icon-document-checked"></i>
          <span slot="title">合同审查</span>
        </el-menu-item>
        <el-menu-item index="/contract-history">
          <i class="el-icon-time"></i>
          <span slot="title">审查历史</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container class="app-main">
      <router-view />
    </el-container>
  </el-container>
</template>

<script>
export default {
  name: 'MainLayout',
  data() {
    return {
      isCollapsed: false
    }
  },
  computed: {
    activeMenu() {
      return this.$route.path
    }
  },
  methods: {
    toggleCollapse() {
      this.isCollapsed = !this.isCollapsed
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
</style>
