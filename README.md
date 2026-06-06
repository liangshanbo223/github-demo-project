# 🚀 new_s-ui：现代网络协议管理面板 (定制增强版)

`new_s-ui` 是一款基于原版 `s-ui` 深度定制与重构的轻量级网络多协议管理面板。本项目旨在提供**极致的安全性**、**极简的证书管理流程**以及**更加现代化的前端交互体验**。

---

## 🌟 核心优化项 (为什么选择 new_s-ui)

### 1. ⚡️ ACME 一键自动化证书系统
* **告别命令行**：不再需要登录终端手动运行繁琐的 `acme.sh` 脚本。
* **全自动申请与续签**：直接在面板管理页面中填入您的“域名”与“邮箱”，面板将自动通过 HTTP-01 验证挑战，申请并部署 SSL 证书。
* **后台自动巡检**：内置自动续签机制，证书即将过期时自动后台更新，确保 TLS 服务永不断线。

### 2. 🛡️ 内置 CA 根证书生成与一键下载
* **私有证书链保障**：内置 CA 根证书生成器，允许您直接生成专属于您的私有根证书。
* **独家一键下载**：新增内置下载路由 `https://您的域名/ca/download`。客户端只需访问此链接即可下载 `ca.crt`。
* **无缝绿锁信任**：将根证书导入设备（PC、Mac、手机）受信任证书列表后，自签的 TLS 链接即可在客户端实现完美受信任与安全绿锁。

### 3. 🎨 重构的前端 UI 与现代交互交互
* **极致视觉反馈**：基于现代前端框架（Vuetify 3 / Vite）大幅优化，重构了登录界面、状态提示以及响应式适配。
* **人性化操作模态框**：
  * 重构了入站协议配置（Inbounds）模态框，支持 Hysteria 2、TUIC、VLESS 等配置项的直观修改。
  * VLESS (Reality) 密钥支持一键随机生成，减少配置负担。

### 4. ⚙️ 重构的底层设置与数据保护
* **配置零损坏保护**：重构了底层的配置保存与更新机制（`setting.go`），优化了数据库写入逻辑，避免了传统面板因突然断电或重启导致配置损坏、甚至无法进入面板的问题。
* **高可用多网卡监听**：优化了多网卡绑定及路由转发的兼容性，保证高并发状态下的连接稳定性。

---

## 🛠️ 支持的协议与服务

本面板深度集成并完美支持以下主流传输协议与服务：

* **支持协议**：Hysteria 2, TUIC, VLESS (Reality/TLS), VMess, Trojan, Shadowsocks, Socks, Http, Direct 等。
* **传输层**：TCP, gRPC, WebSocket, HTTPUpgrade 等。
* **高级功能**：
  * 实时的系统状态监控（CPU、内存、硬盘、网速占用）。
  * 独立的客户端多租户流量统计、限速、有效期限制。
  * 便捷的面板数据一键备份与恢复。

---

## 📦 一键安装与部署

### 快速安装 (推荐)
在您的 Linux 服务器（支持 Ubuntu, Debian, Centos）上运行以下一键脚本即可完成部署：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/liangshanbo223/github-demo-project/main/install.sh)
```

### 常用命令
面板安装成功后，您可以在终端中使用 `s-ui` 快捷命令进行管理：

```bash
s-ui             # 打开管理菜单 (包含重启、停止、密码重置、端口修改等)
s-ui start       # 启动面板
s-ui stop        # 停止面板
s-ui restart     # 重启面板
s-ui status      # 查看当前运行状态
```

---

## 📂 项目结构一览

```bash
github-demo-project/
├── main.go               # 项目入口
├── install.sh            # 一键安装脚本
├── go.mod                # 依赖管理
├── api/                  # 面板 API 控制器层
├── database/             # 数据库模型与初始化 (SQLite)
├── service/              # 证书管理 (ACME / CA)、入站控制、用户管理核心服务
├── frontend/             # 基于 Vuetify/TypeScript 构建的前端源码
├── web/                  # 后端 Gin Web 框架与路由配置
└── windows/              # Windows 系统的服务包与本地调试运行脚本
```

---

## 🤝 参与开发与贡献

如果您发现了 Bug 或有任何功能上的建议，欢迎通过 GitHub 提交 [Issue](https://github.com/liangshanbo223/github-demo-project/issues) 或 [Pull Request](https://github.com/liangshanbo223/github-demo-project/pulls)。

*感谢所有为原项目及衍生定制版做出贡献的开发者！*
