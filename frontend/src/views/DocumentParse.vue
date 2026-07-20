<template>
  <div class="document-page">
    <div class="page-header">
      <h2>📄 文档转换</h2>
      <p class="page-desc">上传文档并转换为目标格式</p>
    </div>

    <el-tabs v-model="activeMode" class="mode-tabs">
      <el-tab-pane label="格式转换" name="convert">
        <el-card class="upload-card">
          <el-upload
            :key="uploadKey"
            ref="upload"
            drag
            action=""
            :auto-upload="false"
            :show-file-list="false"
            :on-change="handleFileChange"
            :on-remove="handleFileRemove"
            :file-list="fileList"
          >
            <i class="el-icon-upload"></i>
            <div class="el-upload__text">将文件拖到此处，或<em>点击选择</em></div>
            <div slot="tip" class="el-upload__tip">支持PDF、Word、图片等多种格式，单文件不超过 50MB</div>
          </el-upload>
        </el-card>
        <el-card v-if="fileList.length" class="file-list-card">
          <div v-for="f in fileList" :key="f.id" class="file-item">
            <span class="file-name">{{ f.name }}</span>
            <span class="file-size">{{ toKB(f.size) }}</span>
            <el-button type="text" size="mini" icon="el-icon-delete" @click="removeFile(f)" style="color:#999;"/>
          </div>
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

          <div class="action-bar" style="text-align: center;">
            <el-button type="primary" :loading="converting" :disabled="!selectedFile" @click="handleConvert">
              {{ converting ? '转换中...' : '开始转换' }}
            </el-button>
          </div>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="JSON 比对" name="jsonCompare">
        <div class="compare-layout">
          <div class="compare-inputs">
            <el-card class="compare-card">
              <div slot="header">JSON A <el-tag size="mini" type="danger">旧</el-tag></div>
              <el-input type="textarea" v-model="jsonA" :rows="12" placeholder="粘贴或输入第一个 JSON..." class="json-input" />
            </el-card>
            <el-card class="compare-card">
              <div slot="header">JSON B <el-tag size="mini" type="success">新</el-tag></div>
              <el-input type="textarea" v-model="jsonB" :rows="12" placeholder="粘贴或输入第二个 JSON..." class="json-input" />
            </el-card>
          </div>
          <div class="compare-action">
            <el-button type="primary" :loading="comparing" :disabled="!jsonA || !jsonB" @click="handleJsonCompare">
              {{ comparing ? '比对中...' : '开始比对' }}
            </el-button>
          </div>
          <div v-if="compareResult" class="compare-result">
            <el-card>
              <div slot="header">
                比对结果
                <el-tag v-if="compareResult.match" type="success" size="mini" style="margin-left:8px">完全一致</el-tag>
                <el-tag v-else type="warning" size="mini" style="margin-left:8px">{{ compareResult.differences.length }} 处差异</el-tag>
              </div>
              <div v-if="!compareResult.match" class="diff-list">
                <div v-for="(d, i) in compareResult.differences" :key="i" class="diff-item" :class="'diff-' + d.type">
                  <div class="diff-path">{{ d.path }}</div>
                  <div class="diff-type">
                    <el-tag v-if="d.type==='added'" type="success" size="mini">新增</el-tag>
                    <el-tag v-else-if="d.type==='removed'" type="danger" size="mini">删除</el-tag>
                    <el-tag v-else type="warning" size="mini">修改</el-tag>
                  </div>
                  <div class="diff-values">
                    <div v-if="d.old_value !== undefined" class="diff-old">
                      <span class="diff-label">旧值：</span><code>{{ formatValue(d.old_value) }}</code>
                    </div>
                    <div v-if="d.new_value !== undefined" class="diff-new">
                      <span class="diff-label">新值：</span><code>{{ formatValue(d.new_value) }}</code>
                    </div>
                  </div>
                </div>
              </div>
            </el-card>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="MD 转 Word" name="md2docx">
        <el-card class="upload-card">
          <el-upload
            ref="mdUpload"
            drag
            action=""
            :auto-upload="false"
            :show-file-list="false"
            :on-change="handleMdFileChange"
            :on-remove="handleMdFileRemove"
            :file-list="mdFileList"
            :accept="'.md'"
          >
            <i class="el-icon-upload"></i>
            <div class="el-upload__text">将 .md 文件拖到此处，或<em>点击选择</em></div>
            <div slot="tip" class="el-upload__tip">支持 Markdown（.md）格式，单文件不超过 20MB</div>
          </el-upload>
        </el-card>
        <el-card v-if="mdFileList.length" class="file-list-card">
          <div v-for="f in mdFileList" :key="f.id" class="file-item">
            <span class="file-name">{{ f.name }}</span>
            <span class="file-size">{{ toKB(f.size) }}</span>
            <el-button type="text" size="mini" icon="el-icon-delete" @click="removeMdFile(f)" style="color:#999;"/>
          </div>
        </el-card>

        <el-card class="options-card">
          <div class="action-bar" style="text-align: center;">
            <el-button type="primary" :loading="mdConverting" :disabled="!mdSelectedFile" @click="handleMdConvert">
              {{ mdConverting ? '转换中...' : '开始转换' }}
            </el-button>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

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
import { convertFile, mdFile2DocxFile, jsonCompare } from '@/api/document'

export default {
  name: 'DocumentParse',
  data() {
    return {
      activeMode: 'convert',
      // mode 1: format conversion
      uploadKey: 0,
      fileList: [],
      selectedFile: null,
      toFormats: 'md',
      doOcr: true,
      converting: false,
      // mode 2: md to docx
      mdFileList: [],
      mdSelectedFile: null,
      mdConverting: false,
      // shared result
      resultBlob: null,
      resultFileName: '',
      // mode 3: json compare
      jsonA: '',
      jsonB: '',
      comparing: false,
      compareResult: null,
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
    toKB(bytes) {
      if (!bytes) return '0 KB'
      return (bytes / 1024).toFixed(2) + ' KB'
    },
    removeFile(file) {
      this.fileList = this.fileList.filter(f => f.id !== file.id)
      if (this.selectedFile && this.selectedFile.name === file.name) {
        this.selectedFile = null
      }
    },
    removeMdFile(file) {
      this.mdFileList = this.mdFileList.filter(f => f.id !== file.id)
      if (this.mdSelectedFile && this.mdSelectedFile.name === file.name) {
        this.mdSelectedFile = null
      }
    },
    // === mode 1: format conversion ===
    handleFileRemove() {
      this.selectedFile = null
      this.fileList = []
      this.resultBlob = null
      this.resultFileName = ''
    },
    handleFileChange(file) {
      const isValidType = this.checkFileType(file);
      if (!isValidType) {
        this.$message.error('仅支持 PDF、Word 文档和图片格式！');
        return;
      }
      if (file.size > 50 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 50MB！');
        return;
      }
      if (this.fileList.length > 0) {
        this.$message.error('仅支持单个文件！');
        return;
      }
      this.fileList = [file]
      this.uploadKey++
      this.selectedFile = file.raw
      this.resultBlob = null
      this.resultFileName = ''
    },
    checkFileType(file) {
      if (this.allowedTypes.includes(file.raw.type)) return true
      const fileName = file.name.toLowerCase()
      return this.allowedExtensions.some(ext => fileName.endsWith(ext))
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
              const errData = JSON.parse(reader.result)
              this.$message.error(errData.message || errData.msg || '转换失败')
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
    // === mode 2: md to docx ===
    handleMdFileRemove() {
      this.mdSelectedFile = null
      this.mdFileList = []
      this.resultBlob = null
      this.resultFileName = ''
    },
    handleMdFileChange(file) {
      if (!/\.md$/i.test(file.name)) {
        this.$message.error('仅支持 .md 格式文件')
        return
      }
      if (file.size > 20 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 20MB')
        return
      }
      if (this.mdFileList.length > 0) {
        this.$message.error('仅支持单个文件！')
        return
      }
      this.mdFileList = [file]
      this.mdSelectedFile = file.raw
      this.resultBlob = null
      this.resultFileName = ''
    },
    async handleMdConvert() {
      if (!this.mdSelectedFile) return
      this.mdConverting = true
      try {
        const res = await mdFile2DocxFile(this.mdSelectedFile)
        this.resultBlob = res.data
        const baseName = this.mdSelectedFile.name.replace(/\.md$/i, '')
        this.resultFileName = `${baseName}.docx`
        this.mdSelectedFile = null
        this.mdFileList = []
        this.$refs.mdUpload.clearFiles()
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
        this.mdConverting = false
      }
    },
    // === shared ===
    // === mode 3: json compare ===
    async handleJsonCompare() {
      if (!this.jsonA || !this.jsonB) return
      this.comparing = true
      try {
        const res = await jsonCompare(this.jsonA, this.jsonB)
        this.compareResult = res.data.data || res.data
      } catch (err) {
        this.$message.error(err.response?.data?.msg || err.message || '比对失败')
      } finally {
        this.comparing = false
      }
    },
    formatValue(v) {
      if (v === null) return 'null'
      if (v === undefined) return ''
      if (typeof v === 'object') return JSON.stringify(v, null, 2)
      return String(v)
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
.file-item { display: flex; align-items: center; justify-content: space-between; padding: 4px 0; font-size: 13px; color: #333; }
.file-name { flex: 1; }
.file-size { flex: 1; color: #999; font-size: 12px; text-align: left; }
.file-list-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.file-list-card >>> .el-card__body { padding: 6px 16px; }
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
.options-card,
.result-card {
  border-radius: 0;
  border-top: 0;
  margin-bottom: 0;
}
.upload-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.upload-card >>> .el-upload-dragger { margin-bottom: 0; width: 100%; }
.upload-card >>> .el-card__body { padding: 12px 16px; }
.upload-card >>> .el-upload { width: 100%; }
.upload-card >>> .el-upload-dragger { width: 100%; padding: 16px; }
.options-card >>> .el-card__body {
  padding: 20px 32px;
}
.result-card >>> .el-card__body {
  padding: 16px 32px;
}
.mode-tabs { padding: 0 32px; background: #fff; }
.mode-tabs >>> .el-tabs__header { margin-bottom: 0; }
.action-bar {
  padding-top: 8px;
}
.compare-layout { padding: 16px 0; }
.compare-inputs { display: flex; gap: 16px; }
.compare-card { flex: 1; }
.compare-card >>> .el-card__header { font-weight: 600; font-size: 14px; }
.json-input >>> textarea { font-family: 'Courier New', monospace; font-size: 13px; }
.compare-action { text-align: center; padding: 16px 0; }
.compare-result { margin-top: 0; }
.diff-list { max-height: 500px; overflow-y: auto; }
.diff-item { padding: 10px 12px; border-bottom: 1px solid #f0f0f0; }
.diff-item:last-child { border-bottom: none; }
.diff-path { font-family: 'Courier New', monospace; font-size: 13px; color: #409eff; font-weight: 600; margin-bottom: 4px; }
.diff-type { margin-bottom: 4px; }
.diff-values { font-size: 13px; }
.diff-label { color: #999; }
.diff-old code { color: #f56c6c; background: #fef0f0; }
.diff-new code { color: #67c23a; background: #f0f9eb; }
.diff-values code { display: inline-block; padding: 1px 6px; border-radius: 3px; font-family: 'Courier New', monospace; font-size: 12px; white-space: pre-wrap; max-width: 100%; word-break: break-all; }
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
