<template>
  <v-container>
    <!-- 扫描器主体卡片 -->
    <v-card class="rounded-xl mb-4" elevation="4">
      <v-card-title class="d-flex align-center pt-4">
        <v-icon icon="mdi-radar" class="mr-2" color="primary" size="large"></v-icon>
        <span class="font-weight-bold text-h5">Reality 伪装站极速扫描器</span>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        提供“大厂知名域名一键测速校验”与“公网IP段TLS启发式扫描”双重引擎，帮助您在 3 秒内找到最完美、低延迟且洁净的 Reality 伪装站点。
      </v-card-subtitle>

      <v-tabs v-model="tab" color="primary" class="px-4">
        <v-tab value="validate">
          <v-icon start icon="mdi-flash-fast" color="success"></v-icon>
          大厂域名一键校验 (极速推荐)
        </v-tab>
        <v-tab value="scan">
          <v-icon start icon="mdi-dns-outline" color="primary"></v-icon>
          公网IP段深度探测 (启发式)
        </v-tab>
      </v-tabs>

      <v-divider></v-divider>

      <v-card-text class="pt-4">
        <v-window v-model="tab">
          <!-- 模式 A：大厂域名一键校验 -->
          <v-window-item value="validate">
            <v-row>
              <v-col cols="12">
                <v-textarea
                  v-model="testDomains"
                  label="待测速校验的全球大厂域名列表 (逗号或换行分隔)"
                  rows="4"
                  variant="outlined"
                  persistent-placeholder
                  hint="已为您预置了 20+ 个最优质的 Reality 备选域名。支持直接编辑修改。"
                  persistent-hint
                  class="mb-4"
                ></v-textarea>
              </v-col>
            </v-row>
            <v-row class="mt-0">
              <v-col cols="12" sm="6">
                <v-text-field
                  v-model.number="timeout"
                  label="TLS 握手超时时间 (秒)"
                  type="number"
                  min="1"
                  max="10"
                  variant="outlined"
                  hide-details
                ></v-text-field>
              </v-col>
              <v-col cols="12" sm="6" class="d-flex align-center justify-end">
                <v-btn
                  color="success"
                  variant="flat"
                  size="large"
                  prepend-icon="mdi-rocket-launch"
                  :loading="isValidating"
                  :disabled="isValidating"
                  @click="startValidation"
                  class="rounded-lg px-6"
                >
                  开始一键测速校验
                </v-btn>
              </v-col>
            </v-row>
          </v-window-item>

          <!-- 模式 B：IP 段深度扫描 -->
          <v-window-item value="scan">
            <v-row>
              <v-col cols="12" md="6">
                <v-select
                  v-model="selectedPreset"
                  :items="presets"
                  label="快速选择扫描目标段"
                  variant="outlined"
                  hide-details
                  @update:modelValue="onPresetChange"
                  class="mb-4"
                ></v-select>
              </v-col>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="serverIpDisplay"
                  label="当前服务器公网 IP"
                  readonly
                  variant="outlined"
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
                  label="公网扫描 IP 对象 (💡 重点推荐扫描当前主机同 C 段网段以实现超低延迟邻近伪装)"
                  placeholder="例如: 12.34.56.0/24。无需扫描外部各种 CDN，建议优先扫描当前主机同网段"
                  rows="3"
                  variant="outlined"
                  :disabled="selectedPreset !== 'custom'"
                  class="mb-2"
                ></v-textarea>
                <div class="text-caption text-primary mb-4 font-weight-medium">
                  💡 邻近网段扫描最为关键！我们强烈建议优先针对您当前服务器所在的同 C 段或同 B 段进行扫描。由于物理邻近，同网段不仅握手延迟最低，还可以使您的 Reality 伪装在网络拓扑上显得最为自然真实。
                </div>
              </v-col>
            </v-row>
            <v-row>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="heuristicSni"
                  label="启发式 SNI 域名 (Heuristic SNI Fallback)"
                  variant="outlined"
                  placeholder="例如: yahoo.com"
                  hint="遭遇 CDN/虚拟主机无 SNI 握手被拒时，自动以此域名重新尝试 TLS 协商以解析真实证书"
                  persistent-hint
                  class="mb-4"
                ></v-text-field>
              </v-col>
              <v-col cols="12" md="6">
                <v-row>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="threads"
                      label="并发线程"
                      type="number"
                      min="10"
                      max="1000"
                      variant="outlined"
                      hide-details
                    ></v-text-field>
                  </v-col>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="timeout"
                      label="超时(秒)"
                      type="number"
                      min="1"
                      max="10"
                      variant="outlined"
                      hide-details
                    ></v-text-field>
                  </v-col>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="duration"
                      label="限时(秒)"
                      type="number"
                      min="10"
                      max="600"
                      variant="outlined"
                      hide-details
                    ></v-text-field>
                  </v-col>
                </v-row>
              </v-col>
            </v-row>
            <v-row>
              <v-col cols="12" class="d-flex justify-end pt-2">
                <v-btn
                  v-if="!scanStatus.is_running"
                  color="primary"
                  variant="flat"
                  size="large"
                  prepend-icon="mdi-play"
                  @click="startScan"
                  class="rounded-lg px-6"
                >
                  开始并发深度扫描
                </v-btn>
                <template v-else>
                  <v-btn
                    v-if="!scanStatus.is_paused"
                    color="warning"
                    variant="flat"
                    size="large"
                    prepend-icon="mdi-pause"
                    @click="pauseScan"
                    class="rounded-lg px-6 mr-2"
                  >
                    暂停扫描
                  </v-btn>
                  <v-btn
                    v-else
                    color="success"
                    variant="flat"
                    size="large"
                    prepend-icon="mdi-play"
                    @click="resumeScan"
                    class="rounded-lg px-6 mr-2"
                  >
                    继续扫描
                  </v-btn>
                  <v-btn
                    color="error"
                    variant="flat"
                    size="large"
                    prepend-icon="mdi-stop"
                    @click="stopScan"
                    class="rounded-lg px-6"
                  >
                    停止扫描任务
                  </v-btn>
                </template>
              </v-col>
            </v-row>
          </v-window-item>
        </v-window>
      </v-card-text>
    </v-card>

    <!-- 运行状态与进度卡片 (仅当 IP 段扫描运行时显示) -->
    <v-card v-if="tab === 'scan' && (scanStatus.is_running || scanStatus.total > 0)" class="rounded-xl mb-4" elevation="4">
      <v-card-text>
        <v-row align="center">
          <v-col cols="12" md="4" class="text-h6 d-flex align-center">
            <v-icon icon="mdi-pulse" class="mr-2" :color="scanStatus.is_running ? (scanStatus.is_paused ? 'warning' : 'success') : 'grey'"></v-icon>
            状态: {{ scanStatus.is_running ? (scanStatus.is_paused ? '扫描已暂停' : '正在并发扫描 IP 段中...') : '扫描已结束' }}
          </v-col>
          <v-col cols="12" md="4" class="text-subtitle-1 text-center">
            已探测进度: {{ scanStatus.scanned }} / {{ scanStatus.total }} IP 
            ({{ Math.round((scanStatus.scanned / (scanStatus.total || 1)) * 100) }}%)
          </v-col>
          <v-col cols="12" md="4" class="text-subtitle-1 text-right text-primary font-weight-bold">
            已捕获优质伪装站: {{ scanStatus.found_count }} 个
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

    <!-- 校验与探测结果卡片 -->
    <v-card class="rounded-xl" elevation="4">
      <v-card-title class="d-flex align-center pt-4">
        <v-icon icon="mdi-list-status" class="mr-2" color="primary"></v-icon>
        <span class="font-weight-bold">
          {{ tab === 'validate' ? '全球大厂域名校验结果 (已排序)' : 'Reality 公网扫描发现列表' }}
        </span>
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
          <!-- 模式 A：大厂域名校验的表格内容 -->
          <template v-if="tab === 'validate'">
            <tr v-if="isValidating">
              <td colspan="7" class="text-center py-8">
                <v-progress-circular indeterminate color="success" size="40" class="mb-2"></v-progress-circular>
                <div class="text-grey subtitle-1">大厂域名本地握手测速及 HTTP 校验中，请稍候 (约需 3 秒)...</div>
              </td>
            </tr>
            <tr v-else-if="validationResults.length === 0">
              <td colspan="7" class="text-center text-grey py-8">
                暂无域名校验结果。请在上方输入域名列表并点击“开始一键测速校验”。
              </td>
            </tr>
            <tr v-for="item in validationResults" :key="item.ip + '-' + item.domain" :class="{'highlight-fastest': item === validationResults[0]}">
              <td class="font-monospace">
                {{ item.ip }}
                <v-chip v-if="item === validationResults[0]" size="x-small" color="success" class="ml-1" variant="flat">极速推荐</v-chip>
              </td>
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
                  color="success"
                  size="small"
                  variant="flat"
                  prepend-icon="mdi-check-bold"
                  @click="applyResult(item)"
                >
                  一键应用配置
                </v-btn>
              </td>
            </tr>
          </template>

          <!-- 模式 B：IP 段扫描发现的表格内容 -->
          <template v-else>
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
                  一键应用配置
                </v-btn>
              </td>
            </tr>
          </template>
        </tbody>
      </v-table>
    </v-card>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted, inject, Ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { useRouter } from 'vue-router'

const loading: Ref = inject('loading') ?? ref(false)
const router = useRouter()

const tab = ref('validate')
const isValidating = ref(false)
const validationResults = ref<any[]>([])

// 预置 20+ 个最优质的全球 Reality 备选大厂域名
const testDomains = ref([
  'yahoo.com',
  'apple.com',
  'microsoft.com',
  'samsung.com',
  'nvidia.com',
  'ikea.com',
  'amazon.com',
  'cloudflare.com',
  'speedtest.net',
  'cnn.com',
  'github.com',
  'mapbox.com',
  'docker.com',
  'zoom.us',
  'nytimes.com',
  'ebay.com',
  'target.com',
  'puma.com',
  'netflix.com',
  'visa.com',
  'booking.com',
  'steamcommunity.com',
  'microsoftonline.com',
  'adobe.com',
  'oracle.com'
].join(', '))

const serverIp = ref('')
const serverIpDisplay = ref('获取中...')
const targets = ref('')
const threads = ref(100)
const timeout = ref(3)
const duration = ref(300)
const heuristicSni = ref('yahoo.com') // 默认使用 yahoo.com 作为启发式 SNI

const selectedPreset = ref('host_c') // 默认围绕当前主机 IP 段展开
const presets = ref([
  { title: '当前主机同 C 段 (/24) (🚀 强烈推荐)', value: 'host_c' },
  { title: '当前主机同 B 段抽样 (/16)', value: 'host_b' },
  { title: 'Cloudflare 节点段 (104.16.0.0/16)', value: 'cloudflare' },
  { title: 'Cloudflare 备用段 (104.18.0.0/16)', value: 'cloudflare_alt' },
  { title: 'Fastly 节点段 (151.101.0.0/16)', value: 'fastly' },
  { title: 'Amazon CloudFront 段 (13.224.0.0/16)', value: 'cloudfront' },
  { title: '自定义输入段/域名', value: 'custom' },
])

const scanStatus = ref({
  is_running: false,
  is_paused: false,
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
    
    if (selectedPreset.value === 'host_c') {
      targets.value = cSegment
    }
    
    presets.value.forEach(p => {
      if (p.value === 'host_c') {
        p.title = `当前主机同 C 段 (${cSegment}) (🚀 强烈推荐)`
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

// 模式 A：一键快速大厂域名校验
const startValidation = async () => {
  if (!testDomains.value.trim()) {
    alert('请输入需要校验的域名')
    return
  }
  isValidating.value = true
  validationResults.value = []
  const res = await HttpUtils.post('api/scanner/validate', {
    domains: testDomains.value,
    timeout: timeout.value
  })
  isValidating.value = false
  if (res.success) {
    validationResults.value = res.obj || []
  }
}

// 模式 B：IP段深度并发扫描
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
    duration: duration.value,
    heuristic_sni: heuristicSni.value // 支持启发式 SNI Fallback
  })
  loading.value = false
  if (res.success) {
    await fetchScanStatus()
  }
}

const pauseScan = async () => {
  loading.value = true
  const res = await HttpUtils.post('api/pauseScan', {})
  loading.value = false
  if (res.success) {
    await fetchScanStatus()
  }
}

const resumeScan = async () => {
  loading.value = true
  const res = await HttpUtils.post('api/resumeScan', {})
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
.highlight-fastest {
  background-color: rgba(76, 175, 80, 0.08) !important;
  transition: background-color 0.3s ease;
}
</style>
