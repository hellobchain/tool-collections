<template>
  <el-dialog title="📌 上周遗留事项" :visible.sync="visible" :close-on-click-modal="false" :close-on-press-escape="false" width="500px">
    <p class="dialog-desc">以下是你上周标记为"待完成"的事项，请选择本周是否继续跟进：</p>
    <div class="carryover-list">
      <div v-for="item in carryover" :key="item.id" class="carryover-item">
        <el-checkbox v-model="selectedMap[item.id]">
          {{ item.content }}
        </el-checkbox>
      </div>
    </div>
    <span slot="footer">
      <el-button @click="handleCancel">全部舍弃</el-button>
      <el-button type="primary" @click="handleConfirm">确认选择</el-button>
    </span>
  </el-dialog>
</template>

<script>
export default {
  props: {
    carryover: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      visible: true,
      selectedMap: {},
      confirming: false
    }
  },
  watch: {
    carryover: {
      immediate: true,
      handler(val) {
        const map = {}
        val.forEach(item => {
          map[item.id] = true  // 默认全部保留
        })
        this.selectedMap = map
      }
    }
  },
  methods: {
    handleConfirm() {
      const kept = []
      const dropped = []
      this.carryover.forEach(item => {
        if (this.selectedMap[item.id]) {
          kept.push(item.id)
        } else {
          dropped.push(item.id)
        }
      })
      this.visible = false
      this.$emit('confirm', { kept, dropped })
    },
    handleCancel() {
      const dropped = this.carryover.map(item => item.id)
      this.visible = false
      this.$emit('confirm', { kept: [], dropped })
    }
  }
}
</script>

<style scoped>
.dialog-desc {
  color: #666;
  margin-bottom: 16px;
}
.carryover-item {
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}
</style>