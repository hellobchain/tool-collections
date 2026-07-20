<template>
  <el-popover trigger="hover" placement="top" :width="popoverWidth">
    <div slot="reference" class="ellipsis-cell" :style="{ maxWidth }">
      {{ content }}
    </div>
    <div class="ellipsis-full">{{ content }}</div>
  </el-popover>
</template>

<script>
export default {
  name: 'EllipsisCell',
  props: {
    content: { type: [String, Number], default: '' },
    maxWidth: { type: String, default: '300px' },
    popoverWidth: { type: [Number, String], default: 400 }
  }
}
</script>

<style scoped>
.ellipsis-cell {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  width: 100%;
  vertical-align: middle;
}
.ellipsis-full {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
  max-height: 400px;
  overflow-y: auto;
}
</style>
