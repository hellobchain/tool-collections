<template>
  <div class="json-tool-page">
    <div class="page-header">
      <h2>{ } JSON 在线工具</h2>
      <p class="page-desc">格式化、压缩、验证、转义、转换 JSON 数据</p>
    </div>

    <div class="tool-layout">
      <div class="input-section">
        <el-card class="input-card">
          <div slot="header" class="card-header">
            <span>输入</span>
            <div class="header-controls">
              <el-button size="mini" @click="loadExample">Demo</el-button>
              <el-button size="mini" @click="showHistory = true">历史</el-button>
              <el-button size="mini" type="danger" plain @click="clearAll">清空</el-button>
            </div>
          </div>
          <div class="editor-wrapper">
            <div class="line-numbers" ref="inputLines">
              <div v-for="n in inputLineCount" :key="n" class="line-number">{{ n }}</div>
            </div>
            <textarea
              ref="jsonInput"
              class="json-editor"
              v-model="inputText"
              placeholder="在此输入或粘贴 JSON 数据..."
              @input="onInput"
              @scroll="syncInputScroll"
              spellcheck="false"
            ></textarea>
          </div>
          <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
        </el-card>

        <div class="toolbar">
          <div class="toolbar-row">
            <el-button-group>
              <el-button size="small" type="primary" @click="formatJson">格式化</el-button>
              <el-button size="small" @click="compressJson">压缩</el-button>
              <el-button size="small" @click="validateJson">验证</el-button>
            </el-button-group>
            <el-button-group>
              <el-button size="small" @click="escapeJson">转义</el-button>
              <el-button size="small" @click="unescapeJson">反转义</el-button>
            </el-button-group>
          </div>
          <div class="toolbar-row">
            <el-button-group>
              <el-button size="small" @click="toUnicode">中文→Unicode</el-button>
              <el-button size="small" @click="fromUnicode">Unicode→中文</el-button>
            </el-button-group>
            <el-button-group>
              <el-button size="small" @click="toGetParam">转 Get 参数</el-button>
              <el-button size="small" @click="fromGetParam">Get 参数→JSON</el-button>
            </el-button-group>
            <el-button size="small" @click="toggleDictJson">Dict↔JSON</el-button>
          </div>
        </div>
      </div>

      <div class="output-section">
        <el-card class="output-card">
          <div slot="header" class="card-header">
            <span>输出</span>
            <div class="header-controls">
              <el-button size="mini" @click="toggleTreeView">{{ treeVisible ? '关闭树形' : '树形编辑' }}</el-button>
              <el-select v-model="langTarget" size="mini" placeholder="编程语言转换" style="width:150px" @change="handleLangConvert">
                <el-option label="编程语言转换" value="" disabled />
                <el-option label="Go" value="go" />
                <el-option label="Java" value="java" />
                <el-option label="Python" value="python" />
                <el-option label="TypeScript" value="typescript" />
                <el-option label="Rust" value="rust" />
                <el-option label="Swift" value="swift" />
                <el-option label="C++" value="cpp" />
                <el-option label="C#" value="csharp" />
                <el-option label="Kotlin" value="kotlin" />
                <el-option label="PHP" value="php" />
              </el-select>
              <el-button size="mini" type="primary" @click="copyOutput">复制</el-button>
            </div>
            <div class="header-settings">
              <el-checkbox v-model="showOutputLineNumbers">行号</el-checkbox>
              <span class="setting-label">缩进</span>
              <el-select v-model="indentSize" size="mini" style="width:70px" @change="reformatOutput">
                <el-option label="1" :value="1" />
                <el-option label="2" :value="2" />
                <el-option label="3" :value="3" />
                <el-option label="4" :value="4" />
              </el-select>
              <span class="setting-label">字号</span>
              <el-select v-model="fontSize" size="mini" style="width:70px">
                <el-option label="12" :value="12" />
                <el-option label="13" :value="13" />
                <el-option label="14" :value="14" />
                <el-option label="15" :value="15" />
                <el-option label="16" :value="16" />
                <el-option label="18" :value="18" />
                <el-option label="20" :value="20" />
              </el-select>
            </div>
          </div>
          <div class="editor-wrapper">
            <div v-if="showOutputLineNumbers" class="line-numbers" ref="outputLines">
              <div v-for="n in outputLineCount" :key="n" class="line-number">{{ n }}</div>
            </div>
            <textarea
              v-show="!treeVisible"
              ref="jsonOutput"
              class="json-editor"
              :style="{ fontSize: fontSize + 'px' }"
              v-model="outputText"
              readonly
              spellcheck="false"
            ></textarea>
            <div v-show="treeVisible" class="tree-view" ref="treeView"></div>
          </div>
        </el-card>

        <div class="status-bar">
          <span>字符: {{ String(outputText || '').length }}</span>
          <span>行数: {{ outputLineCount }}</span>
          <span>状态: <el-tag :type="statusType" size="mini">{{ statusText }}</el-tag></span>
        </div>
      </div>
    </div>

    <el-dialog title="历史记录" :visible.sync="showHistory" width="500px">
      <div v-if="historyList.length === 0" style="text-align:center;color:#999;padding:20px">暂无记录</div>
      <div v-else class="history-list">
        <div v-for="(item, i) in historyList" :key="i" class="history-item" @click="restoreHistory(item)">
          <div class="history-preview">{{ item.slice(0, 200) }}</div>
          <el-button size="mini" type="danger" plain @click.stop="removeHistory(i)">删除</el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { langConvert } from '@/api/json_tool'

const HISTORY_KEY = 'jsontool_history'
const MAX_HISTORY = 50

export default {
  name: 'JsonTool',
  data() {
    return {
      inputText: '',
      outputText: '',
      errorMsg: '',
      indentSize: 2,
      fontSize: 14,
      showOutputLineNumbers: true,
      treeVisible: false,
      langTarget: '',
      langConverting: false,
      showHistory: false,
      historyList: []
    }
  },
  computed: {
    inputLineCount() {
      return ((this.inputText || '').match(/\n/g) || []).length + 1
    },
    outputLineCount() {
      return ((this.outputText || '').match(/\n/g) || []).length + 1
    },
    statusType() {
      if (!this.inputText.trim()) return 'info'
      try {
        JSON.parse(this.inputText)
        return 'success'
      } catch {
        return 'danger'
      }
    },
    statusText() {
      if (!this.inputText.trim()) return '就绪'
      try {
        JSON.parse(this.inputText)
        return '有效 JSON'
      } catch (e) {
        return '无效 JSON: ' + e.message
      }
    }
  },
  watch: {
    fontSize(val) {
      this.$nextTick(() => {
        if (this.$refs.jsonOutput) {
          this.$refs.jsonOutput.style.fontSize = val + 'px'
        }
      })
    }
  },
  mounted() {
    this.loadHistory()
  },
  methods: {
    onInput() {
      this.errorMsg = ''
      this.langTarget = ''
    },
    syncInputScroll() {
      if (this.$refs.inputLines) {
        this.$refs.inputLines.scrollTop = this.$refs.jsonInput.scrollTop
      }
    },
    syncOutputScroll() {
      if (this.$refs.outputLines) {
        this.$refs.outputLines.scrollTop = this.$refs.jsonOutput.scrollTop
      }
    },

    getParsed() {
      try {
        return JSON.parse(this.inputText)
      } catch (e) {
        this.errorMsg = 'JSON 解析失败: ' + e.message
        return null
      }
    },

    formatJson() {
      const obj = this.getParsed()
      if (!obj) return
      this.outputText = JSON.stringify(obj, null, this.indentSize)
      this.saveHistory(this.inputText)
    },

    compressJson() {
      const obj = this.getParsed()
      if (!obj) return
      this.outputText = JSON.stringify(obj)
      this.saveHistory(this.inputText)
    },

    validateJson() {
      const obj = this.getParsed()
      if (!obj) return
      this.outputText = '✓ JSON 格式有效\n\n' + JSON.stringify(obj, null, this.indentSize)
    },

    escapeJson() {
      this.outputText = JSON.stringify(this.inputText)
      this.saveHistory(this.inputText)
    },

    unescapeJson() {
      try {
        var result = JSON.parse('"' + this.inputText.replace(/^"|"$/g, '') + '"')
        this.outputText = String(result)
      } catch {
        try {
          var result2 = JSON.parse(this.inputText)
          this.outputText = typeof result2 === 'object' ? JSON.stringify(result2, null, this.indentSize) : String(result2)
        } catch (e) {
          this.errorMsg = '反转义失败: ' + e.message
        }
      }
      this.saveHistory(this.inputText)
    },

    toUnicode() {
      let result = ''
      for (const ch of this.inputText) {
        if (ch.charCodeAt(0) > 127) {
          result += '\\u' + ch.charCodeAt(0).toString(16).padStart(4, '0')
        } else {
          result += ch
        }
      }
      this.outputText = result
      this.saveHistory(this.inputText)
    },

    fromUnicode() {
      try {
        this.outputText = this.inputText.replace(/\\u([0-9a-fA-F]{4})/g, (_, hex) =>
          String.fromCharCode(parseInt(hex, 16))
        )
      } catch (e) {
        this.errorMsg = 'Unicode 解码失败: ' + e.message
      }
      this.saveHistory(this.inputText)
    },

    toGetParam() {
      const obj = this.getParsed()
      if (!obj) return
      if (typeof obj !== 'object' || Array.isArray(obj)) {
        this.errorMsg = '仅支持对象类型转换'
        return
      }
      const params = new URLSearchParams()
      for (const [k, v] of Object.entries(obj)) {
        params.append(k, String(v))
      }
      this.outputText = params.toString()
      this.saveHistory(this.inputText)
    },

    fromGetParam() {
      try {
        const params = new URLSearchParams(this.inputText)
        const obj = {}
        for (const [k, v] of params) {
          obj[k] = v
        }
        this.outputText = JSON.stringify(obj, null, this.indentSize)
      } catch (e) {
        this.errorMsg = '解析失败: ' + e.message
      }
      this.saveHistory(this.inputText)
    },

    toggleDictJson() {
      const obj = this.getParsed()
      if (!obj) return
      if (typeof obj === 'object' && !Array.isArray(obj)) {
        const lines = Object.entries(obj).map(([k, v]) => `"${k}": "${v}"`)
        this.outputText = '{\n  ' + lines.join(',\n  ') + '\n}'
      } else {
        this.outputText = JSON.stringify(obj, null, this.indentSize)
      }
      this.saveHistory(this.inputText)
    },

    toggleTreeView() {
      this.treeVisible = !this.treeVisible
      if (this.treeVisible) {
        this.buildTree()
      }
    },

    buildTree() {
      this.$nextTick(() => {
        const el = this.$refs.treeView
        if (!el) return
        const obj = this.getParsed()
        if (!obj) {
          el.innerHTML = '<div style="color:#999;padding:16px">无效 JSON，无法展示树形</div>'
          return
        }
        el.innerHTML = this.renderTree(obj, '$')
        el.querySelectorAll('.tree-toggle').forEach(btn => {
          btn.addEventListener('click', () => {
            const children = btn.parentElement.nextElementSibling
            if (children) {
              const isHidden = children.style.display === 'none'
              children.style.display = isHidden ? '' : 'none'
              btn.textContent = isHidden ? '▼' : '▶'
            }
          })
        })
      })
    },

    renderTree(val, path, depth = 0) {
      const indent = '  '.repeat(depth)
      if (val === null) return `<div class="tree-node" style="padding-left:${depth*20}px"><span class="tree-null">null</span></div>`
      if (typeof val !== 'object') {
        const cls = typeof val === 'string' ? 'tree-string' : typeof val === 'boolean' ? 'tree-bool' : 'tree-number'
        const display = typeof val === 'string' ? '"' + val + '"' : String(val)
        return `<div class="tree-node" style="padding-left:${depth*20}px"><span class="${cls}">${this.escapeHtml(display)}</span></div>`
      }
      if (Array.isArray(val)) {
        if (val.length === 0) return `<div class="tree-node" style="padding-left:${depth*20}px"><span class="tree-bracket">[]</span></div>`
        let html = `<div class="tree-node" style="padding-left:${depth*20}px"><span class="tree-toggle">▼</span><span class="tree-bracket">[${val.length}]</span></div>`
        html += `<div class="tree-children">`
        for (let i = 0; i < val.length; i++) {
          html += `<div class="tree-key">${i}:</div>` + this.renderTree(val[i], `${path}[${i}]`, depth + 1)
        }
        html += `</div>`
        return html
      }
      const keys = Object.keys(val)
      if (keys.length === 0) return `<div class="tree-node" style="padding-left:${depth*20}px"><span class="tree-bracket">{}</span></div>`
      let html = `<div class="tree-node" style="padding-left:${depth*20}px"><span class="tree-toggle">▼</span><span class="tree-bracket">{${keys.length}}</span></div>`
      html += `<div class="tree-children">`
      for (const k of keys) {
        html += `<div class="tree-key" style="padding-left:${(depth+1)*20}px">"${this.escapeHtml(k)}": </div>`
          + this.renderTree(val[k], `${path}.${k}`, depth + 1)
      }
      html += `</div>`
      return html
    },

    escapeHtml(s) {
      return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    },

    async handleLangConvert(lang) {
      if (!lang) return
      if (!this.inputText.trim()) return
      this.langConverting = true
      try {
        const res = await langConvert(this.inputText, lang)
        this.outputText = String(res.data.data.code || '')
        this.saveHistory(this.inputText)
      } catch {
        this.$message.error('语言转换失败')
      } finally {
        this.langConverting = false
        this.langTarget = ''
      }
    },

    reformatOutput() {
      if (!this.outputText) return
      if (typeof this.outputText !== 'string') {
        this.outputText = String(this.outputText)
        return
      }
      try {
        const obj = JSON.parse(this.outputText)
        this.outputText = JSON.stringify(obj, null, this.indentSize)
      } catch {
        // not valid JSON output, do nothing
      }
    },

    copyOutput() {
      if (!this.outputText) return
      navigator.clipboard.writeText(this.outputText).then(() => {
        this.$message.success('已复制')
      }).catch(() => {
        this.$refs.jsonOutput.select()
        document.execCommand('copy')
        this.$message.success('已复制')
      })
    },

    clearAll() {
      this.inputText = ''
      this.outputText = ''
      this.errorMsg = ''
      this.treeVisible = false
      this.langTarget = ''
    },

    loadExample() {
      this.inputText = JSON.stringify({
        name: "JSON 工具",
        version: "1.0.0",
        description: "在线 JSON 格式化、压缩、验证、转换",
        features: ["格式化", "压缩", "验证", "转义", "语言转换"],
        config: {
          indent: 2,
          theme: "light",
          lang: "zh-CN"
        },
        active: true,
        count: 42
      }, null, 2)
      this.onInput()
    },

    saveHistory(text) {
      if (!text.trim()) return
      let history = this.getStoredHistory()
      history = history.filter(h => h !== text)
      history.unshift(text)
      if (history.length > MAX_HISTORY) {
        history = history.slice(0, MAX_HISTORY)
      }
      localStorage.setItem(HISTORY_KEY, JSON.stringify(history))
      this.historyList = history
    },

    getStoredHistory() {
      try {
        return JSON.parse(localStorage.getItem(HISTORY_KEY)) || []
      } catch {
        return []
      }
    },

    loadHistory() {
      this.historyList = this.getStoredHistory()
    },

    restoreHistory(text) {
      this.inputText = text
      this.onInput()
      this.showHistory = false
    },

    removeHistory(i) {
      this.historyList.splice(i, 1)
      localStorage.setItem(HISTORY_KEY, JSON.stringify(this.historyList))
    }
  }
}
</script>

<style scoped>
.json-tool-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f7fa;
}
.page-header {
  background: #fff;
  padding: 16px 32px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
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
.tool-layout {
  flex: 1;
  display: flex;
  gap: 16px;
  padding: 16px 32px;
  overflow: hidden;
}
.input-section,
.output-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.input-card,
.output-card {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.input-card >>> .el-card__body,
.output-card >>> .el-card__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.header-controls {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.header-settings {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  flex-wrap: wrap;
}
.setting-label {
  font-size: 12px;
  color: #999;
}
.editor-wrapper {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}
.line-numbers {
  width: 40px;
  background: #f8f9fa;
  border-right: 1px solid #e4e7ed;
  overflow: hidden;
  text-align: right;
  padding: 8px 6px 8px 0;
  user-select: none;
  flex-shrink: 0;
}
.line-number {
  font-size: 12px;
  line-height: 1.5;
  color: #999;
  font-family: 'Courier New', monospace;
}
.json-editor {
  flex: 1;
  border: none;
  outline: none;
  resize: none;
  padding: 8px 12px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  color: #333;
  background: #fff;
  tab-size: 2;
  white-space: pre;
  overflow: auto;
}
.json-editor::placeholder {
  color: #c0c4cc;
}
.toolbar {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-top: none;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex-shrink: 0;
}
.toolbar-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.error-msg {
  color: #f56c6c;
  font-size: 12px;
  padding: 4px 12px;
  background: #fef0f0;
  border-top: 1px solid #fde2e2;
  flex-shrink: 0;
}
.status-bar {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-top: none;
  padding: 4px 12px;
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: #999;
  flex-shrink: 0;
}
.tree-view {
  flex: 1;
  padding: 8px 12px;
  overflow: auto;
  font-family: 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.8;
}
.tree-node {
  white-space: nowrap;
}
.tree-toggle {
  cursor: pointer;
  color: #409eff;
  margin-right: 4px;
  user-select: none;
  font-size: 10px;
}
.tree-key {
  color: #881391;
  display: inline;
}
.tree-string { color: #67c23a; }
.tree-number { color: #409eff; }
.tree-bool { color: #e6a23c; }
.tree-null { color: #999; }
.tree-bracket { color: #333; }
.tree-children { padding-left: 0; }
.history-list {
  max-height: 400px;
  overflow-y: auto;
}
.history-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
}
.history-item:hover {
  background: #f5f7fa;
}
.history-preview {
  flex: 1;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #666;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
