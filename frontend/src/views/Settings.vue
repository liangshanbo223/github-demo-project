<template>
  <v-card :loading="loading">
    <v-tabs
    v-model="tab"
    color="primary"
    align-tabs="center"
    show-arrows
  >
    <v-tab value="t1">{{ $t('setting.interface') }}</v-tab>
    <v-tab value="t2">{{ $t('setting.sub') }}</v-tab>
    <v-tab value="t3">{{ $t('setting.jsonSub') }}</v-tab>
    <v-tab value="t4">{{ $t('setting.clashSub') }}</v-tab>
    <v-tab value="t5">系统维护</v-tab>
    <v-tab value="t6">节点管理</v-tab>
  </v-tabs>
  <v-card-text>
    <v-row align="center" justify="center" style="margin-bottom: 10px;">
      <v-col cols="auto">
        <v-btn color="primary" @click="save" :loading="loading" :disabled="!stateChange">
          {{ $t('actions.save') }}
        </v-btn>
      </v-col>
      <v-col cols="auto">
        <v-btn variant="outlined" color="warning" @click="restartApp" :loading="loading" :disabled="stateChange">
          {{ $t('actions.restartApp') }}
        </v-btn>
      </v-col>
    </v-row>
    <v-window v-model="tab">
      <v-window-item value="t1">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webListen" :label="$t('setting.addr')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model.number="webPort" min="1" type="number" :label="$t('setting.port')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webPath" :label="$t('setting.webPath')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webDomain" :label="$t('setting.domain')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webKeyFile" :label="$t('setting.sslKey')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webCertFile" :label="$t('setting.sslCert')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webURI" :label="$t('setting.webUri')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="sessionMaxAge"
              min="0"
              :label="$t('setting.sessionAge')"
              :suffix="$t('date.m')"
              hide-details
              ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="trafficAge"
              min="0"
              :label="$t('setting.trafficAge')"
              :suffix="$t('date.d')"
              hide-details
              ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.timeLocation" :label="$t('setting.timeLoc')" hide-details></v-text-field>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t2">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" v-model="subEncode" :label="$t('setting.subEncode')" hide-details />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" v-model="subShowInfo" :label="$t('setting.subInfo')" hide-details />
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subListen" :label="$t('setting.addr')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subPort"
              min="1"
              :label="$t('setting.port')"
              hide-details></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subKeyFile" :label="$t('setting.sslKey')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subCertFile" :label="$t('setting.sslCert')" hide-details></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subDomain" :label="$t('setting.domain')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subPath" :label="$t('setting.path')" hide-details></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subUpdates"
              min="0"
              :label="$t('setting.update')"
              hide-details
              ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subURI" :label="$t('setting.subUri')" hide-details></v-text-field>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t3">
        <SubJsonExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t4">
        <SubClashExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t5">
        <v-row>
          <v-col cols="12" md="6">
            <v-card class="rounded-xl" elevation="2" title="系统更新">
              <v-card-text>
                <v-text-field v-model="updateUrl" label="系统更新包下载 URL (.tar.gz 或二进制)" placeholder="https://github.com/liangshanbo223/github-demo-project/releases/..."></v-text-field>
              </v-card-text>
              <v-card-actions>
                <v-btn color="primary" variant="flat" :loading="updating" @click="triggerUpdate">
                  一键平滑升级
                </v-btn>
              </v-card-actions>
            </v-card>
          </v-col>
          <v-col cols="12" md="6">
            <v-card class="rounded-xl" elevation="2" title="版本备份与回滚">
              <v-card-text>
                <div v-if="backups.length === 0">暂无版本备份</div>
                <v-list v-else density="compact">
                  <v-list-item v-for="b in backups" :key="b" :title="b"></v-list-item>
                </v-list>
              </v-card-text>
              <v-card-actions>
                <v-btn color="warning" variant="flat" :loading="rolling" :disabled="backups.length === 0" @click="triggerRollback">
                  回滚到上个版本
                </v-btn>
              </v-card-actions>
            </v-card>
          </v-col>
        </v-row>

        <v-row class="mt-4">
          <v-col cols="12" md="6">
            <v-card class="rounded-xl" elevation="2" title="网络诊断工具箱">
              <v-card-text>
                <v-text-field v-model="diagTarget" label="目标域名、IP或URL" placeholder="baidu.com"></v-text-field>
                <div class="d-flex flex-wrap gap-2 mt-2">
                  <v-btn color="primary" class="mr-2 mb-2" :loading="diagLoading" @click="runDiagnose('ping')">Ping</v-btn>
                  <v-btn color="primary" class="mr-2 mb-2" :loading="diagLoading" @click="runDiagnose('traceroute')">Traceroute</v-btn>
                  <v-btn color="primary" class="mr-2 mb-2" :loading="diagLoading" @click="runDiagnose('curl')">Curl</v-btn>
                  <v-btn color="primary" class="mr-2 mb-2" :loading="diagLoading" @click="runDiagnose('dns')">DNS 解析</v-btn>
                </div>
                <v-textarea v-model="diagOutput" label="诊断结果" readonly class="mt-4" rows="8" variant="outlined" font-family="monospace"></v-textarea>
              </v-card-text>
            </v-card>
          </v-col>
          <v-col cols="12" md="6">
            <v-card class="rounded-xl" elevation="2" title="系统日志实时查看">
              <v-card-text>
                <div class="d-flex justify-space-between align-center mb-2">
                  <span>实时获取 systemd 日志 (s-ui)</span>
                  <v-btn :color="logSource ? 'error' : 'success'" variant="tonal" size="small" @click="toggleLogStream">
                    {{ logSource ? '停止监听' : '开始监听' }}
                  </v-btn>
                </div>
                <v-textarea v-model="sysLogsText" label="系统日志流" readonly rows="8" variant="outlined" font-family="monospace"></v-textarea>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t6">
        <v-card class="rounded-xl" elevation="2">
          <v-card-title class="d-flex justify-space-between align-center">
            <span>分布式多节点列表</span>
            <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="openNodeModal('new')">
              添加节点
            </v-btn>
          </v-card-title>
          <v-card-text>
            <v-table density="comfortable">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>节点名称</th>
                  <th>节点地址</th>
                  <th>同步密钥 (Token)</th>
                  <th>最后心跳</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="node in nodes" :key="node.id">
                  <td>{{ node.id }}</td>
                  <td>{{ node.name }}</td>
                  <td>{{ node.address || '-' }}</td>
                  <td>
                    <span class="text-caption font-monospace" style="user-select: all;">{{ node.token }}</span>
                  </td>
                  <td>{{ node.last_heartbeat ? formatTime(node.last_heartbeat) : '无' }}</td>
                  <td>
                    <v-chip :color="isOnline(node) ? 'success' : 'error'" density="compact" variant="flat">
                      {{ isOnline(node) ? '在线' : '离线' }}
                    </v-chip>
                  </td>
                  <td>
                    <v-btn size="small" variant="text" color="primary" class="mr-2" @click="openNodeModal('edit', node)">
                      编辑
                    </v-btn>
                    <v-btn size="small" variant="text" color="warning" class="mr-2" v-if="node.id !== 0" @click="showInstallCommand(node)">
                      安装命令
                    </v-btn>
                    <v-btn size="small" variant="text" color="error" v-if="node.id !== 0" @click="deleteNode(node)">
                      删除
                    </v-btn>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>
      </v-window-item>
    </v-window>
  </v-card-text>
</v-card>

  <!-- 节点编辑/添加 Dialog -->
  <v-dialog v-model="nodeModal" max-width="500">
    <v-card class="rounded-lg">
      <v-card-title>{{ nodeAction === 'new' ? '添加节点' : '编辑节点' }}</v-card-title>
      <v-divider></v-divider>
      <v-card-text>
        <v-text-field v-model="nodeForm.name" label="节点名称" class="mb-4" hide-details></v-text-field>
        <v-text-field v-model="nodeForm.address" label="节点 IP/域名 (可选)" class="mb-4" hide-details></v-text-field>
        <v-text-field 
          v-model="nodeForm.token" 
          label="同步密钥 (Token)" 
          class="mb-4" 
          hide-details
          append-inner-icon="mdi-cached"
          @click:append-inner="generateRandomToken"
        ></v-text-field>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey" variant="text" @click="nodeModal = false">取消</v-btn>
        <v-btn color="primary" variant="flat" @click="saveNode">保存</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- 获取安装指令 Dialog -->
  <v-dialog v-model="installCmdModal" max-width="700">
    <v-card class="rounded-lg">
      <v-card-title>副节点一键安装与注册</v-card-title>
      <v-divider></v-divider>
      <v-card-text>
        <div class="mb-4 text-subtitle-2 text-grey">在副节点服务器上以 root 权限执行以下命令，会自动下载并安装 S-UI 内核，并自动作为子节点运行，同步当前主控的入站和客户端配置：</div>
        <v-text-field v-model="masterHost" label="主控公网地址/IP" placeholder="12.34.56.78" class="mb-4"></v-text-field>
        <v-textarea 
          v-model="computedInstallCommand" 
          label="一键安装并注册指令" 
          readonly 
          rows="4" 
          variant="outlined" 
          class="font-monospace text-caption"
        ></v-textarea>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey" variant="text" @click="installCmdModal = false">关闭</v-btn>
        <v-btn color="primary" variant="flat" @click="copyCommand">复制命令</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { i18n } from '@/locales'
import { Ref, computed, inject, onMounted, onUnmounted, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { FindDiff } from '@/plugins/utils'
import SubJsonExtVue from '@/components/SubJsonExt.vue'
import SubClashExtVue from '@/components/SubClashExt.vue'
import { push } from 'notivue'
import Data from '@/store/modules/data'
const tab = ref("t1")
const loading:Ref = inject('loading')?? ref(false)
const oldSettings = ref({})

const settings = ref({
	webListen: "",
	webDomain: "",
	webPort: "2095",
	webCertFile: "",
	webKeyFile: "",
  webPath: "/app/",
  webURI: "",
	sessionMaxAge: "0",
  trafficAge: "30",
	timeLocation: "Asia/Shanghai",
  subListen: "",
	subPort: "2096",
	subPath: "/sub/",
	subDomain: "",
	subCertFile: "",
	subKeyFile: "",
	subUpdates: "12",
	subEncode: "true",
	subShowInfo: "false",
	subURI: "",
  subJsonExt: "",
  subClashExt: "",
})

onMounted(async () => {
  loading.value = true
  await loadData()
  await loadBackups()
  loading.value = false
})

const loadData = async () => {
  loading.value = true
  const msg = await HttpUtils.get('api/settings')
  loading.value = false
  if (msg.success) {
    setData(msg.obj)
  }
}

const setData = (data: any) => {
  settings.value = data
  oldSettings.value = { ...data }
}

const save = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/save', { object: 'settings', action: 'set', data: JSON.stringify(settings.value) })
  if (msg.success) {
    push.success({
      title: i18n.global.t('success'),
      duration: 5000,
      message: i18n.global.t('actions.set') + " " + i18n.global.t('pages.settings')
    })
    setData(msg.obj.settings)
  }
  loading.value = false
}

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

const restartApp = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/restartApp',{})
  if (msg.success) {
    let url = settings.value.webURI
    if (url !== "") {
      const isTLS = settings.value.webCertFile !== "" || settings.value.webKeyFile !== ""
      url = buildURL(settings.value.webDomain,settings.value.webPort.toString(),isTLS, settings.value.webPath)
    }
    await sleep(3000)
    window.location.replace(url)
  }
  loading.value = false
}

const buildURL = (host: string, port: string, isTLS: boolean, path: string) => {
  if (!host || host.length == 0) host = window.location.hostname
  if (!port || port.length == 0) port = window.location.port

  const protocol = isTLS ? "https:" : "http:"

  if (port === "" || (isTLS && port === "443") || (!isTLS && port === "80")) {
      port = ""
  } else {
      port = `:${port}`
  }

  return `${protocol}//${host}${port}${path}settings`
}

const subEncode = computed({
  get: () => { return settings.value.subEncode == "true" },
  set: (v:boolean) => { settings.value.subEncode = v ? "true" : "false" }
})

const subShowInfo = computed({
  get: () => { return settings.value.subShowInfo == "true" },
  set: (v:boolean) => { settings.value.subShowInfo = v ? "true" : "false" }
})

const webPort = computed({
  get: () => { return settings.value.webPort.length>0 ? parseInt(settings.value.webPort) : 2095 },
  set: (v:number) => { settings.value.webPort = v>0 ? v.toString() : "2095" }
})

const sessionMaxAge = computed({
  get: () => { return settings.value.sessionMaxAge.length>0 ? parseInt(settings.value.sessionMaxAge) : 0 },
  set: (v:number) => { settings.value.sessionMaxAge = v>0 ? v.toString() : "0" }
})

const trafficAge = computed({
  get: () => { return settings.value.trafficAge.length>0 ? parseInt(settings.value.trafficAge) : 0 },
  set: (v:number) => { settings.value.trafficAge = v>0 ? v.toString() : "0" }
})

const subPort = computed({
  get: () => { return settings.value.subPort.length>0 ? parseInt(settings.value.subPort) : 2096 },
  set: (v:number) => { settings.value.subPort = v>0 ? v.toString() : "2096" }
})

const subUpdates = computed({
  get: () => { return settings.value.subUpdates.length>0 ? parseInt(settings.value.subUpdates) : 12 },
  set: (v:number) => { settings.value.subUpdates = v>0 ? v.toString() : "12" }
})

const stateChange = computed(() => {
  return !FindDiff.deepCompare(settings.value,oldSettings.value)
})

const backups = ref<string[]>([])
const updateUrl = ref('')
const updating = ref(false)
const rolling = ref(false)

const loadBackups = async () => {
  const msg = await HttpUtils.get('api/backups')
  if (msg.success) {
    backups.value = msg.obj || []
  }
}

const triggerUpdate = async () => {
  if (!updateUrl.value) {
    push.error({ message: '请输入更新 URL' })
    return
  }
  updating.value = true
  const msg = await HttpUtils.post('api/updateSystem', { url: updateUrl.value })
  updating.value = false
  if (msg.success) {
    push.success({ message: '升级命令已下发，面板正在后台热重载重启...' })
    setTimeout(() => {
      window.location.reload()
    }, 5000)
  }
}

const triggerRollback = async () => {
  rolling.value = true
  const msg = await HttpUtils.post('api/rollbackSystem', null)
  rolling.value = false
  if (msg.success) {
    push.success({ message: '回滚命令已下发，面板正在后台热重载重启...' })
    setTimeout(() => {
      window.location.reload()
    }, 5000)
  }
}

const diagTarget = ref('baidu.com')
const diagLoading = ref(false)
const diagOutput = ref('')

const runDiagnose = async (type: string) => {
  if (!diagTarget.value) {
    push.error({ message: '请输入目标地址' })
    return
  }
  diagLoading.value = true
  diagOutput.value = '诊断执行中，请稍候...'
  const msg = await HttpUtils.post('api/diagnose', { type, target: diagTarget.value })
  diagLoading.value = false
  if (msg.success) {
    diagOutput.value = msg.obj || '执行完成，无输出'
  } else {
    diagOutput.value = '执行失败: ' + msg.msg
  }
}

const sysLogsText = ref('')
const logSource = ref<EventSource | null>(null)

const toggleLogStream = () => {
  if (logSource.value) {
    logSource.value.close()
    logSource.value = null
    sysLogsText.value += '\n[系统] 已停止日志流监听。\n'
  } else {
    sysLogsText.value = '[系统] 正在连接系统日志流...\n'
    logSource.value = new EventSource(`${location.origin}/api/sysLog`)
    logSource.value.onmessage = (event) => {
      sysLogsText.value += event.data + '\n'
      const lines = sysLogsText.value.split('\n')
      if (lines.length > 500) {
        sysLogsText.value = lines.slice(lines.length - 500).join('\n')
      }
    }
    logSource.value.onerror = (err) => {
      sysLogsText.value += '[系统] 日志流连接中断。\n'
      if (logSource.value) {
        logSource.value.close()
        logSource.value = null
      }
    }
  }
}

onUnmounted(() => {
  if (logSource.value) {
    logSource.value.close()
  }
})

// 节点管理业务逻辑
const nodes = computed(() => Data().nodes)
const nodeModal = ref(false)
const nodeAction = ref('new')
const nodeForm = ref({
  id: 0,
  name: '',
  address: '',
  token: '',
  last_heartbeat: 0,
  online: false,
  sync_status: ''
})
const installCmdModal = ref(false)
const activeNode = ref<any>(null)
const masterHost = ref(window.location.hostname)

const openNodeModal = (action: string, node?: any) => {
  nodeAction.value = action
  if (action === 'new') {
    nodeForm.value = {
      id: 0,
      name: '',
      address: '',
      token: '',
      last_heartbeat: 0,
      online: false,
      sync_status: ''
    }
    generateRandomToken()
  } else {
    nodeForm.value = { ...node }
  }
  nodeModal.value = true
}

const generateRandomToken = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let t = ''
  for (let i = 0; i < 32; i++) {
    t += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  nodeForm.value.token = t
}

const saveNode = async () => {
  if (!nodeForm.value.name) {
    push.error({ message: '请输入节点名称' })
    return
  }
  if (!nodeForm.value.token) {
    push.error({ message: '请输入或生成通信 Token' })
    return
  }
  loading.value = true
  const success = await Data().save('nodes', nodeAction.value, nodeForm.value)
  loading.value = false
  if (success) {
    nodeModal.value = false
  }
}

const deleteNode = async (node: any) => {
  if (confirm(`确定要删除节点 "${node.name}" 吗？`)) {
    loading.value = true
    const success = await Data().save('nodes', 'del', node.id)
    loading.value = false
  }
}

const formatTime = (ts: number) => {
  return new Date(ts * 1000).toLocaleString()
}

const isOnline = (node: any) => {
  if (node.id === 0) return true
  const now = Math.floor(Date.now() / 1000)
  return node.last_heartbeat && (now - node.last_heartbeat < 45)
}

const showInstallCommand = (node: any) => {
  activeNode.value = node
  masterHost.value = window.location.hostname
  installCmdModal.value = true
}

const computedInstallCommand = computed(() => {
  if (!activeNode.value) return ''
  const proto = window.location.protocol
  const port = webPort.value
  const host = masterHost.value
  const apiUrl = `${proto}//${host}:${port}/api`
  return `curl -fsSL https://raw.githubusercontent.com/liangshanbo223/github-demo-project/main/install.sh | bash && s-ui node --api ${apiUrl} --node ${activeNode.value.id} --token ${activeNode.value.token}`
})

const copyCommand = () => {
  navigator.clipboard.writeText(computedInstallCommand.value)
  push.success({ message: '一键安装命令已复制到剪贴板！' })
}
</script>
