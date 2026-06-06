# s-ui 项目深度解析与开发笔记

本项目是基于 `sing-box` 深度定制的轻量级、多用户网络代理管理面板。它通过直接在 Go 代码中内嵌 `sing-box` 内核，实现了零子进程开销、无缝流量统计以及精细化连接控制。

本文档是针对 `s-ui` 源码逐行与逐模块分析后的开发笔记，旨在为接下来的魔改开发提供完整的底层逻辑支撑。

---

## 一、 项目整体架构与目录结构

### 1. 架构概览
`s-ui` 的整体技术栈由以下几部分构成：
- **后端核心**：基于 Go 语言开发。使用 **Gin** 框架提供 RESTful Web API；使用 **GORM + SQLite (WAL 模式)** 进行配置、用户和监控数据的持久化。
- **代理内核**：将 **sing-box** 作为 Go 依赖库直接链入，省去了管理独立外部进程（守护进程、信号通信等）的麻烦，所有配置热重载、流量计数器及连接控制均在 Go 内存中直接操作。
- **前端界面**：基于 **Vue 3 + Vuetify 3 + TypeScript**，构建工具为 **Vite**。开发阶段使用 API 反向代理，生产构建时打包为静态 SPA，利用 Go 的 `go:embed` 技术完全嵌入到 Go 二进制中，生成**单文件发布包**。

### 2. 目录结构说明
```text
├── main.go               # 项目主入口。启动应用或分发命令行工具。
├── app/                  # 应用生命周期管理（App 初始化、各模块 Start 与 Stop 编排）。
├── cmd/                  # 命令行工具解析（例如后台修改端口、重置密码等系统指令）。
├── config/               # 全局硬编码常量与环境变量配置（如数据库路径、日志级别等）。
├── database/             # 数据库层
│   ├── db.go             # 数据库初始化、连接池配置及 GORM 表自动迁移。
│   └── model/            # 数据库实体定义（Inbound, Outbound, Client, Stats 等）。
├── core/                 # 核心内核控制器（最关键的底层交互层）
│   ├── box.go            # sing-box 内核的装配器与细粒度生命周期管理。
│   ├── tracker_stats.go  # 流量追踪器：基于 RoutedConnection 的流量统计。
│   ├── tracker_conn.go   # 连接追踪器：维护活跃套接字映射，支持按入站协议强制切断连接。
│   └── register.go       # 将自定义的扩展协议或服务注册至 sing-box。
├── service/              # 业务逻辑服务层
│   ├── config.go         # 拼接数据库配置，生成 sing-box 启动 JSON。
│   ├── client.go         # 代理客户管理（限额、过期、到期封禁、自动重置）。
│   ├── setting.go        # 系统全局配置项存储与提取。
│   └── stats.go          # 监控与流量统计数据的落库服务。
├── cronjob/              # 计划任务
│   ├── cronJob.go        # 调度管理器。
│   └── ...Job.go         # 看门狗健康检查、流量入库、WAL日志清理、到期封禁等具体任务。
├── web/                  # 面板主 Web API 容器（整合 Gin 和前端 go:embed 静态资源）。
├── sub/                  # 独立的节点订阅分发服务器（格式化输出 Clash, V2ray, JSON 等订阅）。
├── network/              # 网络辅助库（包含自适应 HTTP/HTTPS 监听器）。
└── frontend/             # 前端 Vue 3 项目源码。
```

---

## 二、 数据库建模设计：极其巧妙的 JSON 序列化

由于 `sing-box` 本身协议配置项繁多且随时可能升级或增删，若在数据库中为每种协议单独建表，维护成本将极高。
`s-ui` 在 `database/model`（以 `Inbound`、`Outbound`、`Endpoint` 为代表）中采用了一种**“固定核心字段 + 动态 JSON 选项”**的精妙设计：

### 以 `Inbound` 为例的数据库字段抽象
```go
type Inbound struct {
	Id   uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Type string `json:"type" form:"type"`
	Tag  string `json:"tag" form:"tag" gorm:"unique"`

	TlsId uint `json:"tls_id" form:"tls_id"`
	Tls   *Tls `json:"tls" form:"tls" gorm:"foreignKey:TlsId;references:Id"`

	Addrs   json.RawMessage `json:"addrs" form:"addrs"`
	OutJson json.RawMessage `json:"out_json" form:"out_json"`
	Options json.RawMessage `json:"-" form:"-"` // 不直接参与默认的 JSON 输出
}
```

- **自定义 UnmarshalJSON**：
  在反序列化前端传入的数据时，程序会将 `id`, `type`, `tag`, `tls_id` 等核心通用字段提取出来，赋给 `Inbound` 结构体的成员；其余**非通用配置**（如端口、嗅探、特定协议属性）被统统打包并以 `json.RawMessage` 形式存入 `Options` 中存入数据库。
- **自定义 MarshalJSON**：
  当需要输出给 `sing-box` 内核解析时，重写 `MarshalJSON`。它通过反射/Map合并，将核心字段（如 `type`, `tag`）与 `Options` 中的非通用参数合并，无缝输出为一个扁平的单层 JSON。
- **优点**：
  既保证了关系型数据库在关联 TLS 证书（外键 `TlsId`）及主键查找时的效率，又让数据结构对未来新协议的扩展具备极佳的兼容性。

---

## 三、 底层核心组件分析 (`core`)

`core` 目录是 `s-ui` 最具技术含量的部分。它没有直接调用 sing-box 的标准入口，而是将各个管理器打散，手动装配。

### 1. sing-box 细粒度控制与生命周期管理 (`box.go`)
原本 sing-box 默认只提供一体化的启动入口，`s-ui` 为注入自定义监控，手动拆解并实现了 `Box` 结构体：
- 手动按阶段调用底层的 `StartStateInitialize` -> `StartStateStart` -> `StartStatePostStart` -> `StartStateStarted` 来启动 `EndpointManager`, `InboundManager`, `OutboundManager` 等管理器。
- 能够向 sing-box 路由器中注入自定义的流量与连接追踪器：
  ```go
  router.AppendTracker(statsTracker)
  router.AppendTracker(connTracker)
  ```

### 2. 零内存拷贝流量追踪器 (`tracker_stats.go`)
面板需要精确统计入站、出站和代理用户的实时流量，`s-ui` 避开了消耗大量 CPU 的日志解析，而是通过实现 sing-box 的路由跟踪接口（`RoutedConnection` 和 `RoutedPacketConnection`）：
- 当有 TCP 连接建立时，`RoutedConnection` 方法被调用。
- 从上下文提取出 `Inbound`（入站名）、`Outbound`（出口标签）和 `User`（用户标识，如 VMess 的 Email）。
- 检索或为该实体创建线程安全的原子计数器组（`Counter`，底层是 `atomic.Int64`）。
- 使用 sing-box 提供的 `bufio.NewInt64CounterConn` 包装原始套接字。底层的 TCP/UDP 读写会自动累加到对应的原子计数器上。
- 后台任务定期调用 `GetStats()`，利用 `Swap(0)` 在读取流量的瞬间清零，实现高并发、高精度的流量差量计算，并写库。

### 3. 主动连接切断器 (`tracker_conn.go`)
商用多用户面板常遇到需要立即切断特定用户或协议的连接（例如用户流量超标、禁用某个入站规则）：
- 每次路由链接建立时，`tracker_conn` 会为连接分配一个 UUID，并将真实的 `net.Conn` 保存至全局 map。
- 在连接发生 `Read` 或 `Write` 抛出不可恢复的 IO 错误，或者在调用 `Close()` 时，触发 untrack 清除 map 记录。
- 暴露出 `CloseConnByInbound(inbound string)` 方法。当修改或禁用了某个端口（Inbound）时，业务层会立即调用此方法。该方法加锁遍历 map，对属于该 inbound 的全部套接字强行调用 `Close()`。

---

## 四、 业务逻辑层实现 (`service`)

### 1. 全局配置拼装 (`config.go`)
- 数据库里只存了每个入站规则、出站节点、端点的独立条目，`ConfigService` 负责动态组装。
- 它首先从全局 `Setting` 中获取 sing-box 的主干骨架配置（内含 Log、DNS、Route 路由分流规则），然后再将提取出的 `Inbounds`、`Outbounds`、`Services` 和 `Endpoints` 序列化成数组，填入其中，融合成一个符合 sing-box 标准定义的最终 JSON 字节流，传给内核执行热重载。
- `Save` 方法高度事务化（`tx.Begin()`）。在保存修改时，如果是修改了特定 `clients`（客户端用户），它不会傻傻地重启整个内核，而是找到被修改客户归属的 `inboundIds`，调用 `RestartInbounds(tx, inboundIds)`，仅仅热重载和切断这几个特定端口的连接，实现了**业务不中断热重载**。

### 2. 完善的多用户商业流量包逻辑 (`client.go`)
- **首次连接起算（延迟激活）**：如果设置了 `delayStart = true`，当检测到该用户的累计流量大于 0 时，说明其已激活，自动写入到期时间 `Expiry = time.Now() + ResetDays`，并解除 `delayStart` 限制。
- **周期自动重置流量**：在定时任务中检索 `next_reset < time.Now()` 的用户，自动将上一周期产生的 `Up` 和 `Down` 流量归档至 `TotalUp`/`TotalDown`，然后清空当月额度。若之前因超流量被停机（`enable = false`），则会自动重新设置为 `enable = true`。
- **过期与超额自动停机**：在 `DepleteClients()` 方法中，用一条 SQL 扫出所有处于启用状态但流量已超标或已过期的客户端，一键禁用并入库审计日志 `model.Changes`，同时返回关联的 Inbound ID，通知服务层重新渲染对应入站以把他们剔除。

---

## 五、 自适应 HTTPS 与安全性设计 (`network`)

Web 服务 (`web.go`) 与 订阅服务 (`sub.go`) 都引入了一个非常有意思的监听器包装：
```go
listener = network.NewAutoHttpsListener(listener)
listener = tls.NewListener(listener, c)
```
- **自适应兼容 (Auto HTTPS Listener)**：
  该监听器能够主动分析客户端请求发送的第一个字节（探针）。如果是 TLS 握手特征，就交给 TLS 服务去处理握手，建立 HTTPS 连接；如果只是普通的明文 HTTP 头，则当作明文 HTTP 转发给 Gin。
- **安全防探测**：
  这种设计使得同一个端口在配置证书时，能同时响应 HTTP 和 HTTPS 请求，不会因为“向 HTTPS 端口发送明文 HTTP 请求”而爆出特征显著的 `400 Bad Request (The plain HTTP request was sent to HTTPS port)` 错误。极大地提高了面板在公网隐藏的安全性，规避了指纹探测。

---

## 六、 计划任务调度器 (`cronjob`)

`cronJob.go` 精准维持了整个系统高效健康运行所需的定时齿轮：
1. **`checkCoreJob` (每5秒)**：心跳看门狗。检测 core 运行状态。若崩溃，则自动拉起，保证服务的高可用。
2. **`statsJob` (每10秒)**：拉取内存中的实时流量统计并保存入库。
3. **`depleteJob` (每1分钟)**：扫描并切断流量耗尽/过期的用户。
4. **`WALCheckpointJob` (每10分钟)**：
   由于 SQLite 的高并发 WAL 模式会将写入操作追加到独立的 `-wal` 文件中，如果不主动控制，该文件会不断膨胀导致磁盘读写下降。此任务周期性强制调用 `PRAGMA wal_checkpoint(FULL)`，将 WAL 中的事务安全地合并回主 `.db` 数据库中。
5. **`delStatsJob` (每日)**：清理旧的流量历史数据，防止数据库体积膨胀。

---

## 七、 魔改项目的可行性切入点与建议

根据 `s-ui` 的源码特点，如果您要推出自己的魔改新项目，以下是极具价值的改进方向：

1. **分布式集群化面板 (多节点支持)**：
   - 目前 `s-ui` 是单机一体化面板，后端的 sing-box 只接管本地的入站。
   - **魔改方案**：将 `core` 模块和 `database` 模块剥离。核心数据库只部署在主控机上，各子节点运行精简的 core 代理代理端，通过 gRPC 或者是加密的 Web API 周期性从主控拉取生成的配置，并将各自节点的 RoutedConnection 流量数据周期性上报到主控数据库。
2. **接入三方支付与自动注册逻辑**：
   - 原项目是纯手动配置。
   - **魔改方案**：在 Web 模块引入非登录态可访问的路由（例如 `/register`、`/checkout`），支持对接第三方聚合支付接口。支付回调后，自动调用 `ClientService.Save` 生成对应的配置文件并自动生成订阅。
3. **更精细的流量审计和协议限制**：
   - 纯粹的 IP 或端口阻断是静态的。
   - **魔改方案**：可在自定义路由 `RoutedConnection` 中引入对连接目标（比如特定的 BT 端口、或是特定的阻断域名）的拦截判断。如果发现匹配，可以主动返回阻断或者计数，达到细粒度审计封禁的目的。
