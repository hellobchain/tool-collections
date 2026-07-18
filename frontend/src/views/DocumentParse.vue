<template>
  <div class="document-page">
    <div class="page-header">
      <h2>📄 文档解析</h2>
      <p class="page-desc">上传文档并转换为目标格式</p>
    </div>

    <el-card class="upload-card">
      <el-upload
        :key="uploadKey"
        ref="upload"
        drag
        action=""
        :auto-upload="false"
        :show-file-list="true"
        :on-change="handleFileChange"
        :on-remove="handleFileRemove"
        :file-list="fileList"
      >
        <i class="el-icon-upload"></i>
        <div class="el-upload__text">将文件拖到此处，或<em>点击选择</em></div>
        <div slot="tip" class="el-upload__tip">支持PDF、Word、图片等多种格式，单文件不超过 50MB</div>
      </el-upload>
    </el-card>

    <el-card class="options-card">
      <el-form label-width="120px" size="small">
        <el-form-item label="目标格式">
          <el-select v-model="toFormats" placeholder="选择转换格式">
            <el-option label="Markdown (.md)" value="md" />
            <el-option label="JSON (.json)" value="json" />
            <el-option label="HTML (.html)" value="html" />
            <el-option label="Text (.txt)" value="text" />
          </el-select>
        </el-form-item>
        <el-form-item label="OCR 文字识别">
          <el-switch v-model="doOcr" active-text="开启" inactive-text="关闭" />
        </el-form-item>
      </el-form>

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
import { convertFile } from '@/api/document'

export default {
  name: 'DocumentParse',
  data() {
    return {
      uploadKey: 0,
      fileList: [],
      selectedFile: null,
      toFormats: 'md',
      doOcr: true,
      converting: false,
      resultBlob: null,
      resultFileName: '',
      allowedTypes: [
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'image/jpeg',
      'image/png',
      'image/gif',
      'image/bmp',
      'image/webp'
    ],
      allowedExtensions: ['.pdf', '.doc', '.docx', '.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp']
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
      // 校验文件类型
      const isValidType = this.checkFileType(file);
      if (!isValidType) {
        this.$message.error('仅支持 PDF、Word 文档和图片格式！');
        // 移除不符合的文件
        this.fileList = [];
        this.$refs.upload.clearFiles();
        return;
      }
      // 校验文件大小（50MB）
      if (file.size > 50 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 50MB！');
        this.fileList = [];
        this.$refs.upload.clearFiles();
        return;
      }

      this.fileList = [file]
      this.uploadKey++
      this.selectedFile = file.raw
      this.resultBlob = null
      this.resultFileName = ''
    },
    checkFileType(file) {
      if (this.allowedTypes.includes(file.raw.type)) {
        return true;
      }
      const fileName = file.name.toLowerCase();
      return this.allowedExtensions.some(ext => fileName.endsWith(ext));
    },
    async handleConvert() {
      if (!this.selectedFile || !this.toFormats || !this.fileList.length) return

      this.converting = true
      try {
        const res = await convertFile(this.selectedFile, this.toFormats, this.doOcr)

        const contentType = res.headers['content-type'] || ''
        if (contentType.includes('application/json') && !this.toFormats && !this.toFormats.contains('json')) {
          const reader = new FileReader()
          reader.onload = () => {
            try {
              const err = JSON.parse(reader.result)
              this.$message.error(err.message || err.msg || '转换失败')
            } catch {
              this.$message.error('转换失败')
            }
          }
          reader.readAsText(res.data)
          return
        }

        this.resultBlob = res.data
        const nameParts = this.selectedFile.name.split('.')
        const baseName = nameParts.slice(0, -1).join('.') || this.selectedFile.name
        const extMap = { md: 'md', json: 'json', html: 'html', text: 'txt' }
        this.resultFileName = `${baseName}.${extMap[this.toFormats] || this.toFormats}`
        this.selectedFile = null
        this.fileList = []
        this.$refs.upload.clearFiles()
        this.uploadKey++
        this.$message.success('转换成功')
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
.document-page {
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
.options-card,
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
.options-card >>> .el-card__body {
  padding: 20px 32px;
}
.result-card >>> .el-card__body {
  padding: 16px 32px;
}
.action-bar {
  padding-top: 8px;
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
