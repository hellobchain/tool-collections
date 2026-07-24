<template>
  <div class="portfolio">
    <div class="page-header">
      <h1>持仓管理</h1>
      <div>
        <el-button @click="showCreateAccount = true" icon="el-icon-plus">新建账户</el-button>
        <el-button @click="showRecordTrade = true" type="primary" icon="el-icon-edit">记录交易</el-button>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :span="6" v-for="acc in accounts" :key="acc.id">
        <el-card shadow="hover" class="account-card">
          <div class="acc-name">{{ acc.name }}</div>
          <div class="acc-meta">
            <el-tag size="mini">{{ acc.broker || '手工' }}</el-tag>
            <el-tag size="mini" type="info" v-if="acc.market">{{ acc.market }}</el-tag>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="mt-20">
      <div slot="header">交易记录</div>
      <div class="toolbar">
        <el-input v-model="tradeFilter.symbol" placeholder="股票代码" size="small" class="filter-item" />
        <el-select v-model="tradeFilter.side" placeholder="方向" size="small" class="filter-item" clearable>
          <el-option label="买入" value="buy" />
          <el-option label="卖出" value="sell" />
        </el-select>
        <el-button size="small" type="primary" @click="fetchTrades">查询</el-button>
      </div>
      <el-table :data="trades" stripe>
        <el-table-column prop="trade_date" label="日期" width="100" />
        <el-table-column prop="symbol" label="代码" width="100" />
        <el-table-column prop="side" label="方向" width="60">
          <template slot-scope="s">
            <el-tag :type="s.row.side === 'buy' ? 'success' : 'danger'" size="mini">{{ s.row.side === 'buy' ? '买入' : '卖出' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="quantity" label="数量" width="80" align="right" />
        <el-table-column prop="price" label="价格" width="80" align="right">
          <template slot-scope="s">{{ s.row.price?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column prop="fee" label="手续费" width="80" align="right" />
        <el-table-column prop="market" label="市场" width="60" />
        <el-table-column prop="currency" label="币种" width="60" />
      </el-table>
      <el-empty v-if="!trades.length" description="暂无交易记录" />
    </el-card>

    <el-dialog title="新建账户" :visible.sync="showCreateAccount" width="400px">
      <el-form label-width="90px">
        <el-form-item label="账户名称"><el-input v-model="newAccount.name" /></el-form-item>
        <el-form-item label="券商"><el-input v-model="newAccount.broker" /></el-form-item>
        <el-form-item label="市场"><el-input v-model="newAccount.market" /></el-form-item>
        <el-form-item label="基础币种"><el-select v-model="newAccount.baseCurrency"><el-option label="CNY" value="CNY" /><el-option label="USD" value="USD" /><el-option label="HKD" value="HKD" /></el-select></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="showCreateAccount = false">取消</el-button>
        <el-button type="primary" @click="handleCreateAccount">创建</el-button>
      </span>
    </el-dialog>

    <el-dialog title="记录交易" :visible.sync="showRecordTrade" width="500px">
      <el-form label-width="90px">
        <el-form-item label="账户"><el-select v-model="tradeForm.accountId" filterable><el-option v-for="a in accounts" :key="a.id" :label="a.name" :value="a.id" /></el-select></el-form-item>
        <el-form-item label="股票代码"><el-input v-model="tradeForm.symbol" /></el-form-item>
        <el-form-item label="方向"><el-radio-group v-model="tradeForm.side"><el-radio label="buy">买入</el-radio><el-radio label="sell">卖出</el-radio></el-radio-group></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="tradeForm.quantity" :min="0" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="tradeForm.price" :min="0" :precision="3" /></el-form-item>
        <el-form-item label="手续费"><el-input-number v-model="tradeForm.fee" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="交易日期"><el-date-picker v-model="tradeForm.tradeDate" type="date" value-format="yyyy-MM-dd" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="showRecordTrade = false">取消</el-button>
        <el-button type="primary" @click="handleRecordTrade">记录</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import * as stockApi from '@/api/stock'

export default {
  name: 'Portfolio',
  data() {
    return {
      accounts: [],
      trades: [],
      showCreateAccount: false,
      showRecordTrade: false,
      newAccount: { name: '', broker: '', market: '', baseCurrency: 'CNY' },
      tradeForm: { accountId: '', symbol: '', side: 'buy', quantity: 0, price: 0, fee: 0, tradeDate: '' },
      tradeFilter: { symbol: '', side: '' }
    }
  },
  mounted() { this.fetchAccounts(); this.fetchTrades() },
  methods: {
    async fetchAccounts() {
      try {
        const res = await stockApi.listPortfolioAccounts()
        this.accounts = res.data.data?.accounts || []
      } catch {}
    },
    async fetchTrades() {
      try {
        const res = await stockApi.getPortfolioSnapshot()
        this.trades = []
      } catch {}
    },
    async handleCreateAccount() {
      try {
        await stockApi.createPortfolioAccount(this.newAccount)
        this.$message.success('账户创建成功')
        this.showCreateAccount = false
        this.newAccount = { name: '', broker: '', market: '', baseCurrency: 'CNY' }
        this.fetchAccounts()
      } catch { this.$message.error('创建失败') }
    },
    async handleRecordTrade() {
      try {
        await stockApi.recordPortfolioTrade({
          account_id: this.tradeForm.accountId,
          symbol: this.tradeForm.symbol,
          trade_date: this.tradeForm.tradeDate,
          side: this.tradeForm.side,
          quantity: this.tradeForm.quantity,
          price: this.tradeForm.price,
          fee: this.tradeForm.fee
        })
        this.$message.success('交易已记录')
        this.showRecordTrade = false
      } catch { this.$message.error('记录失败') }
    }
  }
}
</script>

<style scoped>
.portfolio { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h1 { font-size: 22px; margin: 0; }
.account-card { margin-bottom: 10px; cursor: pointer; }
.account-card:hover { transform: translateY(-2px); }
.acc-name { font-size: 16px; font-weight: bold; }
.acc-meta { margin-top: 8px; display: flex; gap: 4px; }
.mt-20 { margin-top: 20px; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.filter-item { width: 140px; }
</style>
