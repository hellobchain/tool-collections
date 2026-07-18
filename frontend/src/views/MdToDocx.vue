<template>
  <div class="md-to-docx-page">
    <div class="page-header">
      <h2>📝 Markdown 转 Word</h2>
      <p class="page-desc">上传 .md 文件，转换为 .docx 格式下载</p>
    </div>

    <el-card class="upload-card">
      <el-upload
        ref="upload"
        drag
        action=""
        :auto-upload="false"
        :show-file-list="true"
        :on-change="handleFileChange"
        :on-remove="handleFileRemove"
        :file-list="fileList"
        :accept="'.md'"
      >
        <i class="el-icon-upload"></i>
        <div class="el-upload__text">将 .md 文件拖到此处，或<em>点击选择</em></div>
        <div slot="tip" class="el-upload__tip">支持 Markdown（.md）格式，单文件不超过 20MB</div>
      </el-upload>
    </el-card>

    <el-card class="action-card">
      <div class="action-bar">
        <el-button type="primary" :loading="converting" :disabled="!selectedFile" @click="handleConvert">
          {{ converting ? '转换中...' : '开始转换' }}
        </el-button>
      </div>
    </el-card>

    <el-card v-if="resultFileName" class="result-card">
      <div class="result-header">
        <i class="el-icon-success" style="color: #67c23a; font-size: 20px;"></i>
        <span>转换完成</span>
      </div>
      <div class="result-actions">
        <el-button type="primary" icon="el-icon-download" @click="handleDownload">下载文件</el-button>
      </div>
    </el-card>
  </div>
</template>

<script>
import { mdFile2DocxFile } from '@/api/document'

export default {
  name: 'MdToDocx',
  data() {
    return {
      fileList: [],
      selectedFile: null,
      converting: false,
      resultBlob: null,
      resultFileName: ''
    }
  },
  methods: {
    handleFileRemove() {
      this.selectedFile = null
      this.fileList = []
      this.resultBlob = null
      this.resultFileName = ''
    },
    handleFileChange(file) {
      if (!/\.md$/i.test(file.name)) {
        this.$message.error('仅支持 .md 格式文件')
        this.fileList = []
        this.$refs.upload.clearFiles()
        return
      }
      if (file.size > 20 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 20MB')
        this.fileList = []
        this.$refs.upload.clearFiles()
        return
      }
      this.fileList = [file]
      this.selectedFile = file.raw
      this.resultBlob = null
      this.resultFileName = ''
    },
    async handleConvert() {
      if (!this.selectedFile || !this.fileList.length) return
      this.converting = true
      try {
        const res = await mdFile2DocxFile(this.selectedFile)
        this.resultBlob = res.data
        const baseName = this.selectedFile.name.replace(/\.md$/i, '')
        this.resultFileName = `${baseName}.docx`
        this.$message.success('转换成功')
        // 转换完成后清空文件选择，防止重复点击
        this.selectedFile = null
        this.fileList = []
        this.$refs.upload.clearFiles()
      } catch (err) {
        if (err.response?.data instanceof Blob) {
          const reader = new FileReader()
          reader.onload = () => {
            try {
              const errData = JSON.parse(reader.result)
              this.$message.error(errData.message || errData.msg || '转换失败')
            } catch {
              this.$message.error('转换失败')
            }
          }
          reader.readAsText(err.response.data)
        } else {
          this.$message.error(err.message || '请求失败')
        }
      } finally {
        this.converting = false
      }
    },
    handleDownload() {
      if (!this.resultBlob) return
      const url = URL.createObjectURL(this.resultBlob)
      const a = document.createElement('a')
      a.href = url
      a.download = this.resultFileName
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  }
}
</script>

<style scoped>
.md-to-docx-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.page-header {
  background: #fff;
  padding: 16px 32px;
  border-bottom: 1px solid #e4e7ed;
}
.page-header h2 {
  font-size: 20px;
  margin: 0 0 4px;
  color: #333;
}
.page-desc {
  color: #999;
  font-size: 14px;
  margin: 0;
}
.upload-card,
.action-card,
.result-card {
  border-radius: 0;
  border-top: 0;
  margin-bottom: 0;
}
.upload-card >>> .el-upload-dragger {
  margin-bottom: 0;
}
.upload-card >>> .el-upload__tip {
  margin-top: 0;
}
.upload-card >>> .el-card__body {
  padding: 20px 32px;
}
.action-card >>> .el-card__body {
  padding: 16px 32px;
}
.result-card >>> .el-card__body {
  padding: 16px 32px;
}
.action-bar {
  padding-top: 0;
}
.result-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  margin-bottom: 12px;
}
.result-actions {
  display: flex;
  gap: 8px;
}
</style>
