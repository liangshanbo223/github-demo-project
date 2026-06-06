<template>
  <v-container>
    <!-- 参数设置卡片 -->
    <v-card class="rounded-xl mb-4" elevation="4">
      <v-card-title class="d-flex align-center">
        <v-icon icon="mdi-radar" class="mr-2" color="primary"></v-icon>
        Reality 伪装站扫描器 (RealiTLScanner)
      </v-card-title>
      <v-card-subtitle>
        本扫描器将在指定的 IP 段中，通过并行探测真实证书并进行 TLS 1.3 握手和页面健康校检，筛选出最纯净、未被污染且适合 Reality 协议的知名大型网站。
      </v-card-subtitle>
      <v-divider class="my-2"></v-divider>
      <v-card-text>
        <v-row>
          <v-col cols="12" md="6">
            <!-- 快速选择 IP 段 -->
            <v-select
              v-model="selectedPreset"
              :items="presets"
              label="快速选择扫描目标段"
              hide-details
              @update:modelValue="onPresetChange"
              class="mb-4"
            ></v-select>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field
              v-model="serverIpDisplay"
              label="本服务器公网 IP"
              readonly
              hide-details
              prepend-inner-icon="mdi-ip-network"
              class="mb-4"
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12">
            <v-textarea
              v-model="targets"
              label="扫描对象 (支持 CIDR、IP 范围、单 IP，多目标换行或逗号分隔)"
              placeholder="例如: 104.16.0.0/16 或者是 151.101.0.0/24"
              rows="3"
              hide-details
              :disabled="selectedPreset !== 'custom'"
              class="mb-4"
            ></v-textarea>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="4">
            <v-text-field
              v-model.number="threads"
              label="并发探测数 (并发线程)"
              type="number"
              min="10"
              max="1000"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field
              v-model.number="timeout"
              label="握手超时时间 (秒)"
              type="number"
              min="1"
              max="10"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field
              v-model.number="duration"
              label="最大扫描时间限制 (秒)"
              type="number"
              min="10"
              max="600"
              hide-details
            ></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
      <v-card-actions class="d-flex justify-end pr-4 pb-4">
        <v-btn
          v-if="!scanStatus.is_running"
          color="primary"
          variant="flat"
          size="large"
          prepend-icon="mdi-play"
          @click="startScan"
        >
          开始并行扫描
        </v-btn>
        <v-btn
          v-else
          color="error"
          variant="flat"
          size="large"
          prepend-icon="mdi-stop"
          @click="stopScan"
        >
          停止扫描
        </v-btn>
      </v-card-actions>
    </v-card>

    <!-- 运行状态与进度卡片 -->
    <v-card v-if="scanStatus.is_running || scanStatus.total > 0" class="rounded-xl mb-4" elevation="4">
      <v-card-text>
        <v-row align="center">
          <v-col cols="12" md="4" class="text-h6 d-flex align-center">
            <v-icon icon="mdi-pulse" class="mr-2" :color="scanStatus.is_running ? 'success' : 'grey'"></v-icon>
            状态: {{ scanStatus.is_running ? '正在深度扫描中...' : '扫描已完成' }}
          </v-col>
          <v-col cols="12" md="4" class="text-subtitle-1 text-center">
            进度: {{ scanStatus.scanned }} / {{ scanStatus.total }} IP 
            ({{ Math.round((scanStatus.scanned / (scanStatus.total || 1)) * 100) }}%)
          </v-col>
          <v-col cols="12" md="4" class="text-subtitle-1 text-right text-primary font-weight-bold">
            已发现伪装网站: {{ scanStatus.found_count }} 个
          </v-col>
        </v-row>
        <v-progress-linear
          color="primary"
          height="10"
          :model-value="(scanStatus.scanned / (scanStatus.total || 1)) * 100"
          striped
          class="rounded mt-2"
        ></v-progress-linear>
      </v-card-text>
    </v-card>

    <!-- 扫描结果列表 -->
    <v-card class="rounded-xl" elevation="4">
      <v-card-title class="d-flex align-center">
        <v-icon icon="mdi-dns-outline" class="mr-2" color="primary"></v-icon>
        扫描出的 Reality 优质目标站列表
      </v-card-title>
      <v-divider></v-divider>
      <v-table hover class="bg-transparent">
        <thead>
          <tr>
            <th>IP 地址</th>
            <th>伪装域名 (SNI)</th>
            <th>ALPN</th>
            <th>TLS 版本</th>
            <th>握手耗时 (延迟)</th>
            <th>HTTP 状态码</th>
            <th class="text-center">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="scanStatus.results.length === 0">
            <td colspan="7" class="text-center text-grey py-8">
              {{ scanStatus.is_running ? '正在探测中，请稍候...' : '暂无扫描结果，请配置上方参数并发起扫描。' }}
            </td>
          </tr>
          <tr v-for="item in scanStatus.results" :key="item.ip + '-' + item.domain">
            <td class="font-monospace">{{ item.ip }}</td>
            <td class="font-weight-bold">{{ item.domain }}</td>
            <td>
              <v-chip size="small" color="secondary" variant="tonal">{{ item.alpn }}</v-chip>
            </td>
            <td>{{ item.tls_version }}</td>
            <td class="font-monospace text-success font-weight-bold">{{ item.delay }} ms</td>
            <td>
              <v-chip size="small" :color="item.status_code < 400 ? 'success' : 'warning'" variant="elevated">
                {{ item.status_code }}
              </v-chip>
            </td>
            <td class="text-center">
              <v-btn
                color="primary"
                size="small"
                variant="flat"
                prepend-icon="mdi-check-bold"
                @click="applyResult(item)"
              >
                一键应用
              </v-btn>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted, inject, Ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'

const loading: Ref = inject('loading') ?? ref(false)
const router = useRouter()

const serverIp = ref('')
const serverIpDisplay = ref('获取中...')
const targets = ref('104.16.0.0/16')
const threads = ref(100)
const timeout = ref(3)
const duration = ref(300)

const selectedPreset = ref('cloudflare')
const presets = ref([
  { title: 'Cloudflare 节点段 (104.16.0.0/16)', value: 'cloudflare' },
  { title: 'Cloudflare 备用段 (104.18.0.0/16)', value: 'cloudflare_alt' },
  { title: 'Fastly 节点段 (151.101.0.0/16)', value: 'fastly' },
  { title: 'Amazon CloudFront 段 (13.224.0.0/16)', value: 'cloudfront' },
  { title: '当前主机同 C 段 (/24)', value: 'host_c' },
  { title: '当前主机同 B 段抽样 (/16)', value: 'host_b' },
  { title: '自定义输入段/域名', value: 'custom' },
])

const scanStatus = ref({
  is_running: false,
  total: 0,
  scanned: 0,
  found_count: 0,
  results: <any[]>[]
})

let timer: any = null

onMounted(async () => {
  await fetchServerIp()
  await fetchScanStatus()
  timer = setInterval(fetchScanStatus, 1500) // 1.5秒轮询一次扫描进度
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})

const fetchServerIp = async () => {
  const res = await HttpUtils.get('api/serverIp')
  if (res.success) {
    serverIp.value = res.obj
    serverIpDisplay.value = res.obj
    
    // 更新主机相关的 presets
    const cSegment = getCSegment(res.obj)
    const bSegment = getBSegment(res.obj)
    
    presets.value.forEach(p => {
      if (p.value === 'host_c') {
        p.title = `当前主机同 C 段 (${cSegment})`
      } else if (p.value === 'host_b') {
        p.title = `当前主机同 B 段抽样 (${bSegment})`
      }
    })
  } else {
    serverIpDisplay.value = '检测失败 (不影响自定义扫描)'
  }
}

const getCSegment = (ip: string): string => {
  const parts = ip.split('.')
  if (parts.length === 4) {
    return `${parts[0]}.${parts[1]}.${parts[2]}.0/24`
  }
  return ip
}

const getBSegment = (ip: string): string => {
  const parts = ip.split('.')
  if (parts.length === 4) {
    return `${parts[0]}.${parts[1]}.0.0/16`
  }
  return ip
}

const onPresetChange = (val: string) => {
  if (val === 'cloudflare') {
    targets.value = '104.16.0.0/16'
  } else if (val === 'cloudflare_alt') {
    targets.value = '104.18.0.0/16'
  } else if (val === 'fastly') {
    targets.value = '151.101.0.0/16'
  } else if (val === 'cloudfront') {
    targets.value = '13.224.0.0/16'
  } else if (val === 'host_c') {
    targets.value = getCSegment(serverIp.value)
  } else if (val === 'host_b') {
    targets.value = getBSegment(serverIp.value)
  } else if (val === 'custom') {
    targets.value = ''
  }
}

const fetchScanStatus = async () => {
  const res = await HttpUtils.get('api/scanStatus')
  if (res.success) {
    scanStatus.value = res.obj
  }
}

const startScan = async () => {
  if (!targets.value.trim()) {
    alert('请输入扫描目标')
    return
  }
  loading.value = true
  const res = await HttpUtils.post('api/startScan', {
    targets: targets.value,
    threads: threads.value,
    timeout: timeout.value,
    duration: duration.value
  })
  loading.value = false
  if (res.success) {
    await fetchScanStatus()
  }
}

const stopScan = async () => {
  loading.value = true
  const res = await HttpUtils.post('api/stopScan', {})
  loading.value = false
  if (res.success) {
    await fetchScanStatus()
  }
}

const applyResult = async (item: any) => {
  loading.value = true
  const res = await HttpUtils.post('api/saveScannerReality', {
    ip: item.ip,
    domain: item.domain
  })
  loading.value = false
  if (res.success) {
    alert(`一键极速配置 Reality 节点成功！\n端口: ${res.obj.port}\n入站标签: ${res.obj.tag}\n客户端 ID (UUID): ${res.obj.uuid}\n\n该配置已部署并生效。点击确定后将前往入站管理查看节点。`)
    router.push('/inbounds')
  }
}
</script>

<style scoped>
.v-table th {
  font-weight: bold !important;
  color: var(--v-theme-primary) !important;
}
.font-monospace {
  font-family: monospace !important;
}
</style>
