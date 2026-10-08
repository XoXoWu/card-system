<template>
  <div class="page">
    <el-row :gutter="16" class="metrics">
      <el-col :span="6" v-for="m in metrics" :key="m.label">
        <div class="metric card-box" :style="{ borderTop: '3px solid ' + m.color }">
          <div class="metric-num" :style="{ color: m.color }">{{ m.value }}</div>
          <div class="metric-label">{{ m.label }}</div>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :span="14">
        <div class="card-box">
          <h4 class="block-title">近 30 天激活趋势（测试卡除外）</h4>
          <div ref="trendRef" class="chart"></div>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="card-box">
          <h4 class="block-title">套餐分布</h4>
          <div ref="pieRef" class="chart"></div>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="24">
        <div class="card-box">
          <el-alert type="info" :closable="false" show-icon
            title="统计口径：测试卡不计入销售 KPI 与激活率；过期为动态计算；封禁卡不计入使用中。" />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import * as echarts from 'echarts'
import http from '../api'

const overview = ref({})
const trendRef = ref()
const pieRef = ref()

const metrics = computed(() => [
  { label: '卡密总数', value: overview.value.total_cards ?? '-', color: '#1d3557' },
  { label: '已激活', value: overview.value.activated ?? '-', color: '#457b9d' },
  { label: '使用中', value: overview.value.in_use ?? '-', color: '#2a9d8f' },
  { label: '今日激活', value: overview.value.today_activated ?? '-', color: '#e63946' },
  { label: '在线设备（15 分钟内）', value: overview.value.online_devices ?? '-', color: '#e76f51' },
  { label: '激活率', value: overview.value.activation_rate != null ? (overview.value.activation_rate * 100).toFixed(1) + '%' : '-', color: '#6d597a' },
  { label: '测试卡发放量', value: overview.value.test_issued ?? '-', color: '#b5838d' },
  { label: '封禁卡数', value: overview.value.banned ?? '-', color: '#6c757d' }
])

onMounted(async () => {
  overview.value = await http.get('/stats/overview')
  const trend = await http.get('/stats/trend')
  const dist = await http.get('/stats/plan-dist')

  const tChart = echarts.init(trendRef.value)
  tChart.setOption({
    grid: { left: 40, right: 16, top: 20, bottom: 24 },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: trend.map(d => d.date) },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'line', smooth: true, data: trend.map(d => d.count), areaStyle: { opacity: .15 }, itemStyle: { color: '#457b9d' } }]
  })

  const pChart = echarts.init(pieRef.value)
  pChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} 张 ({d}%)' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie', radius: ['40%', '65%'], center: ['50%', '45%'],
      label: { formatter: '{b}\n{c} 张' },
      data: dist.map(d => ({ name: d.name, value: d.total }))
    }]
  })
  window.addEventListener('resize', () => { tChart.resize(); pChart.resize() })
})
</script>

<style scoped>
.metrics { margin-bottom: 16px; }
.metric { text-align: center; padding: 22px 12px; }
.metric-num { font-size: 32px; font-weight: 700; }
.metric-label { color: #909399; font-size: 13px; margin-top: 6px; }
.chart { height: 320px; }
.block-title { margin-bottom: 8px; font-size: 14px; color: #303133; }
</style>
