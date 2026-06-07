package api

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/liangshanbo223/github-demo-project/database"
	"github.com/liangshanbo223/github-demo-project/database/model"
	"github.com/liangshanbo223/github-demo-project/logger"
	"github.com/liangshanbo223/github-demo-project/service"
	"github.com/liangshanbo223/github-demo-project/util"
	"github.com/liangshanbo223/github-demo-project/util/common"
	"github.com/gofrs/uuid/v5"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApiService struct {
	service.SettingService
	service.UserService
	service.ConfigService
	service.ClientService
	service.TlsService
	service.InboundService
	service.OutboundService
	service.EndpointService
	service.ServicesService
	service.PanelService
	service.StatsService
	service.ServerService
	service.NodeService
}

func (a *ApiService) LoadData(c *gin.Context) {
	data, err := a.getData(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, nil)
}

func (a *ApiService) getData(c *gin.Context) (interface{}, error) {
	data := make(map[string]interface{}, 0)
	lu := c.Query("lu")
	isUpdated, err := a.ConfigService.CheckChanges(lu)
	if err != nil {
		return "", err
	}
	onlines, err := a.StatsService.GetOnlines()

	sysInfo := a.ServerService.GetSingboxInfo()
	if sysInfo["running"] == false {
		logs := a.ServerService.GetLogs("1", "debug")
		if len(logs) > 0 {
			data["lastLog"] = logs[0]
		}
	}

	if err != nil {
		return "", err
	}
	if isUpdated {
		config, err := a.SettingService.GetConfig()
		if err != nil {
			return "", err
		}
		clients, err := a.ClientService.GetAll()
		if err != nil {
			return "", err
		}
		tlsConfigs, err := a.TlsService.GetAll()
		if err != nil {
			return "", err
		}
		inbounds, err := a.InboundService.GetAll()
		if err != nil {
			return "", err
		}
		outbounds, err := a.OutboundService.GetAll()
		if err != nil {
			return "", err
		}
		endpoints, err := a.EndpointService.GetAll()
		if err != nil {
			return "", err
		}
		services, err := a.ServicesService.GetAll()
		if err != nil {
			return "", err
		}
		nodes, err := a.NodeService.GetAll()
		if err != nil {
			return "", err
		}
		subURI, err := a.SettingService.GetFinalSubURI(getHostname(c))
		if err != nil {
			return "", err
		}
		trafficAge, err := a.SettingService.GetTrafficAge()
		if err != nil {
			return "", err
		}
		data["config"] = json.RawMessage(config)
		data["clients"] = clients
		data["tls"] = tlsConfigs
		data["inbounds"] = inbounds
		data["outbounds"] = outbounds
		data["endpoints"] = endpoints
		data["services"] = services
		data["nodes"] = nodes
		data["subURI"] = subURI
		data["enableTraffic"] = trafficAge > 0
		data["onlines"] = onlines
	} else {
		data["onlines"] = onlines
	}

	return data, nil
}

func (a *ApiService) LoadPartialData(c *gin.Context, objs []string) error {
	data := make(map[string]interface{}, 0)
	id := c.Query("id")

	for _, obj := range objs {
		switch obj {
		case "inbounds":
			inbounds, err := a.InboundService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = inbounds
		case "outbounds":
			outbounds, err := a.OutboundService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = outbounds
		case "endpoints":
			endpoints, err := a.EndpointService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = endpoints
		case "services":
			services, err := a.ServicesService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = services
		case "tls":
			tlsConfigs, err := a.TlsService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = tlsConfigs
		case "clients":
			clients, err := a.ClientService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = clients
		case "config":
			config, err := a.SettingService.GetConfig()
			if err != nil {
				return err
			}
			data[obj] = json.RawMessage(config)
		case "settings":
			settings, err := a.SettingService.GetAllSetting()
			if err != nil {
				return err
			}
			data[obj] = settings
		case "nodes":
			nodes, err := a.NodeService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = nodes
		}
	}

	jsonObj(c, data, nil)
	return nil
}

func (a *ApiService) GetUsers(c *gin.Context) {
	users, err := a.UserService.GetUsers()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, *users, nil)
}

func (a *ApiService) GetSettings(c *gin.Context) {
	data, err := a.SettingService.GetAllSetting()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStats(c *gin.Context) {
	resource := c.Query("resource")
	tag := c.Query("tag")
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		limit = 100
	}
	data, err := a.StatsService.GetStats(resource, tag, limit)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStatus(c *gin.Context) {
	request := c.Query("r")
	result := a.ServerService.GetStatus(request)
	jsonObj(c, result, nil)
}

func (a *ApiService) GetOnlines(c *gin.Context) {
	onlines, err := a.StatsService.GetOnlines()
	jsonObj(c, onlines, err)
}

func (a *ApiService) GetLogs(c *gin.Context) {
	count := c.Query("c")
	level := c.Query("l")
	logs := a.ServerService.GetLogs(count, level)
	jsonObj(c, logs, nil)
}

func (a *ApiService) CheckChanges(c *gin.Context) {
	actor := c.Query("a")
	chngKey := c.Query("k")
	count := c.Query("c")
	changes := a.ConfigService.GetChanges(actor, chngKey, count)
	jsonObj(c, changes, nil)
}

func (a *ApiService) GetKeypairs(c *gin.Context) {
	kType := c.Query("k")
	options := c.Query("o")
	keypair := a.ServerService.GenKeypair(kType, options)
	jsonObj(c, keypair, nil)
}

func (a *ApiService) GetDb(c *gin.Context) {
	exclude := c.Query("exclude")
	db, err := database.GetDb(exclude)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=s-ui_"+time.Now().Format("20060102-150405")+".db")
	c.Writer.Write(db)
}

func (a *ApiService) postActions(c *gin.Context) (string, json.RawMessage, error) {
	var data map[string]json.RawMessage
	err := c.ShouldBind(&data)
	if err != nil {
		return "", nil, err
	}
	return string(data["action"]), data["data"], nil
}

func (a *ApiService) Login(c *gin.Context) {
	remoteIP := getRemoteIp(c)
	loginUser, err := a.UserService.Login(c.Request.FormValue("user"), c.Request.FormValue("pass"), remoteIP)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	sessionMaxAge, err := a.SettingService.GetSessionMaxAge()
	if err != nil {
		logger.Infof("Unable to get session's max age from DB")
	}

	err = SetLoginUser(c, loginUser, sessionMaxAge)
	if err == nil {
		logger.Info("user ", loginUser, " login success")
	} else {
		logger.Warning("login failed: ", err)
	}

	jsonMsg(c, "", nil)
}

func (a *ApiService) ChangePass(c *gin.Context) {
	id := c.Request.FormValue("id")
	oldPass := c.Request.FormValue("oldPass")
	newUsername := c.Request.FormValue("newUsername")
	newPass := c.Request.FormValue("newPass")
	err := a.UserService.ChangePass(id, oldPass, newUsername, newPass)
	if err == nil {
		logger.Info("change user credentials success")
		jsonMsg(c, "save", nil)
	} else {
		logger.Warning("change user credentials failed:", err)
		jsonMsg(c, "", err)
	}
}

func (a *ApiService) Save(c *gin.Context, loginUser string) {
	hostname := getHostname(c)
	obj := c.Request.FormValue("object")
	act := c.Request.FormValue("action")
	data := c.Request.FormValue("data")
	initUsers := c.Request.FormValue("initUsers")
	objs, err := a.ConfigService.Save(obj, act, json.RawMessage(data), initUsers, loginUser, hostname)
	if err != nil {
		jsonMsg(c, "save", err)
		return
	}
	err = a.LoadPartialData(c, objs)
	if err != nil {
		jsonMsg(c, obj, err)
	}
}

func (a *ApiService) RestartApp(c *gin.Context) {
	err := a.PanelService.RestartPanel(3)
	jsonMsg(c, "restartApp", err)
}

func (a *ApiService) RestartSb(c *gin.Context) {
	err := a.ConfigService.RestartCore()
	jsonMsg(c, "restartSb", err)
}

func (a *ApiService) LinkConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, _, err := util.GetOutbound(link, 0)
	jsonObj(c, result, err)
}

func (a *ApiService) SubConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, err := util.GetExternalSub(link)
	jsonObj(c, result, err)
}

func (a *ApiService) ImportDb(c *gin.Context) {
	file, _, err := c.Request.FormFile("db")
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	defer file.Close()
	err = database.ImportDB(file)
	jsonMsg(c, "", err)
}

func (a *ApiService) Logout(c *gin.Context) {
	loginUser := GetLoginUser(c)
	if loginUser != "" {
		logger.Infof("user %s logout", loginUser)
	}
	ClearSession(c)
	jsonMsg(c, "", nil)
}

func (a *ApiService) LoadTokens() ([]byte, error) {
	return a.UserService.LoadTokens()
}

func (a *ApiService) GetTokens(c *gin.Context) {
	loginUser := GetLoginUser(c)
	tokens, err := a.UserService.GetUserTokens(loginUser)
	jsonObj(c, tokens, err)
}

func (a *ApiService) AddToken(c *gin.Context) {
	loginUser := GetLoginUser(c)
	expiry := c.Request.FormValue("expiry")
	expiryInt, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	desc := c.Request.FormValue("desc")
	token, err := a.UserService.AddToken(loginUser, expiryInt, desc)
	jsonObj(c, token, err)
}

func (a *ApiService) DeleteToken(c *gin.Context) {
	tokenId := c.Request.FormValue("id")
	err := a.UserService.DeleteToken(tokenId)
	jsonMsg(c, "", err)
}

func (a *ApiService) GetSingboxConfig(c *gin.Context) {
	rawConfig, err := a.ConfigService.GetConfig("")
	if err != nil {
		c.Status(400)
		c.Writer.WriteString(err.Error())
		return
	}
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=config_"+time.Now().Format("20060102-150405")+".json")
	c.Writer.Write(*rawConfig)
}

func (a *ApiService) GetCheckOutbound(c *gin.Context) {
	tag := c.Query("tag")
	link := c.Query("link")
	result := a.ConfigService.CheckOutbound(tag, link)
	jsonObj(c, result, nil)
}

func (a *ApiService) GetAcmeLogStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	acmeService := service.AcmeService{}
	ch := acmeService.GetAcmeLogChan()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// 清空一下现有通道以保证数据新鲜
	for len(ch) > 0 {
		<-ch
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("message", msg)
			return true
		case <-ticker.C:
			c.SSEvent("ping", "keep-alive")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (a *ApiService) PostAcmeIssue(c *gin.Context) {
	var params struct {
		Email       string   `json:"email" form:"email"`
		Domains     []string `json:"domains" form:"domains"`
		DnsProvider string   `json:"dnsProvider" form:"dnsProvider"`
		DnsParams   []string `json:"dnsParams" form:"dnsParams"`
	}
	if err := c.ShouldBind(&params); err != nil {
		jsonMsg(c, "", err)
		return
	}

	acmeService := service.AcmeService{}

	if acmeService.CheckAcmeRunning() {
		jsonMsg(c, "", common.NewError("ACME 申请任务已在运行中"))
		return
	}

	if params.DnsProvider == "" {
		occupied, process := acmeService.CheckPortOccupied(80)
		if occupied {
			errMsg := fmt.Sprintf("80 端口已被本机的 [%s] 进程占用。HTTP-01 验证必须临时绑定 80 端口以响应 Let's Encrypt 挑战。建议转用 DNS API 申请，或先手动关闭该冲突进程。", process)
			jsonMsg(c, "", common.NewError(errMsg))
			return
		}
	}

	go acmeService.RunAcmeIssue(params.Email, params.Domains, params.DnsProvider, params.DnsParams)

	jsonMsg(c, "success", nil)
}

func (a *ApiService) NodeConfig(c *gin.Context, nodeId uint, token string) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("id = ? AND token = ?", nodeId, token).First(&node).Error
	if err != nil {
		c.JSON(403, gin.H{"status": "failed", "error": "invalid node or token"})
		return
	}

	node.LastHeartbeat = time.Now().Unix()
	node.Online = true
	db.Save(&node)

	configBytes, err := a.ConfigService.GetNodeConfig(nodeId)
	if err != nil {
		c.JSON(500, gin.H{"status": "failed", "error": err.Error()})
		return
	}
	c.Data(200, "application/json", *configBytes)
}

func (a *ApiService) NodeReport(c *gin.Context) {
	var req struct {
		NodeId  uint   `json:"node_id" form:"node_id"`
		Token   string `json:"token" form:"token"`
		Status  struct {
			Cpu float64 `json:"cpu" form:"cpu"`
			Mem float64 `json:"mem" form:"mem"`
		} `json:"status" form:"status"`
		Traffic []struct {
			Tag  string `json:"tag" form:"tag"`
			Up   int64  `json:"up" form:"up"`
			Down int64  `json:"down" form:"down"`
		} `json:"traffic" form:"traffic"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"status": "failed", "error": err.Error()})
		return
	}

	db := database.GetDB()
	var node model.Node
	err := db.Where("id = ? AND token = ?", req.NodeId, req.Token).First(&node).Error
	if err != nil {
		c.JSON(403, gin.H{"status": "failed", "error": "invalid node or token"})
		return
	}

	node.LastHeartbeat = time.Now().Unix()
	node.Online = true
	node.SyncStatus = "synchronized"
	node.Address = fmt.Sprintf("CPU: %.1f%%, MEM: %.1f%%", req.Status.Cpu, req.Status.Mem)
	db.Save(&node)

	tx := db.Begin()
	for _, t := range req.Traffic {
		if t.Up > 0 {
			tx.Model(model.Client{}).Where("name = ?", t.Tag).UpdateColumn("up", gorm.Expr("up + ?", t.Up))
		}
		if t.Down > 0 {
			tx.Model(model.Client{}).Where("name = ?", t.Tag).UpdateColumn("down", gorm.Expr("down + ?", t.Down))
		}
	}
	tx.Commit()

	c.JSON(200, gin.H{"status": "success"})
}

func (a *ApiService) UpdateSystem(c *gin.Context) {
	url := c.Request.FormValue("url")
	if url == "" {
		jsonMsg(c, "", common.NewError("缺少 url 参数"))
		return
	}
	err := a.PanelService.UpdateSystem(url)
	jsonMsg(c, "updateSystem", err)
}

func (a *ApiService) RollbackSystem(c *gin.Context) {
	err := a.PanelService.RollbackSystem()
	jsonMsg(c, "rollbackSystem", err)
}

func (a *ApiService) GetBackups(c *gin.Context) {
	backups, err := a.PanelService.ListBackups()
	jsonObj(c, backups, err)
}

func (a *ApiService) RecoveryPassword(c *gin.Context) {
	key := c.Request.FormValue("key")
	newUser := c.Request.FormValue("username")
	newPass := c.Request.FormValue("password")

	if key == "" || newUser == "" || newPass == "" {
		jsonMsg(c, "", common.NewError("所有参数均为必填项"))
		return
	}

	recoveryKeyDb, err := a.SettingService.GetString("recoveryKey")
	if err != nil || recoveryKeyDb == "" || recoveryKeyDb != key {
		jsonMsg(c, "", common.NewError("Recovery Key 无效或不匹配"))
		return
	}

	// Reset administrative credentials
	err = a.UserService.ChangePass("1", "", newUser, newPass)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	logger.Info("Administrative credentials successfully reset via Recovery Key")
	jsonMsg(c, "success", nil)
}

func (a *ApiService) PostDiagnose(c *gin.Context) {
	var req struct {
		Type   string `json:"type" form:"type"`
		Target string `json:"target" form:"target"`
	}
	if err := c.ShouldBind(&req); err != nil {
		jsonMsg(c, "", err)
		return
	}

	// Validate target to prevent command injection
	for _, r := range req.Target {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '/' || r == ':' || r == '_') {
			jsonMsg(c, "", common.NewError("非法目标地址，仅支持域名、IP或URL"))
			return
		}
	}

	var cmd *exec.Cmd
	switch req.Type {
	case "ping":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ping", "-n", "4", req.Target)
		} else {
			cmd = exec.Command("ping", "-c", "4", req.Target)
		}
	case "traceroute":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("tracert", "-h", "10", req.Target)
		} else {
			cmd = exec.Command("traceroute", "-m", "10", req.Target)
		}
	case "curl":
		cmd = exec.Command("curl", "-Is", "--max-time", "5", req.Target)
	case "dns":
		cmd = exec.Command("nslookup", req.Target)
	default:
		jsonMsg(c, "", common.NewError("不支持的诊断类型"))
		return
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		jsonObj(c, string(output)+"\n错误: "+err.Error(), nil)
		return
	}
	jsonObj(c, string(output), nil)
}

func (a *ApiService) GetSysLogStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	cmd := exec.Command("journalctl", "-u", "s-ui", "-f", "-n", "100")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.SSEvent("error", err.Error())
		return
	}
	if err := cmd.Start(); err != nil {
		// Fallback to streaming internal logs
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Send initial logs
		logs := logger.GetLogs(50, "debug")
		c.SSEvent("message", strings.Join(logs, "\n"))

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ticker.C:
				logs := logger.GetLogs(10, "debug")
				c.SSEvent("message", strings.Join(logs, "\n"))
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
		return
	}

	defer cmd.Process.Kill()

	reader := bufio.NewReader(stdout)

	c.Stream(func(w io.Writer) bool {
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
			} else {
				lineCh <- strings.TrimSpace(line)
			}
		}()

		select {
		case line := <-lineCh:
			c.SSEvent("message", line)
			return true
		case <-errCh:
			return false
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (a *ApiService) PostStartScan(c *gin.Context) {
	var params struct {
		Targets      string `json:"targets" form:"targets"`
		Threads      int    `json:"threads" form:"threads"`
		TimeoutSec   int    `json:"timeout" form:"timeout"`
		DurationSec  int    `json:"duration" form:"duration"`
		HeuristicSni string `json:"heuristic_sni" form:"heuristic_sni"`
	}
	if err := c.ShouldBind(&params); err != nil {
		jsonMsg(c, "", err)
		return
	}

	if params.Threads <= 0 {
		params.Threads = 100
	}
	if params.TimeoutSec <= 0 {
		params.TimeoutSec = 3
	}
	if params.DurationSec <= 0 {
		params.DurationSec = 300
	}

	scanner := service.GetScannerService()
	err := scanner.StartScan(params.Targets, params.Threads, params.TimeoutSec, params.DurationSec, params.HeuristicSni)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	jsonMsg(c, "success", nil)
}

func (a *ApiService) PostValidateDomains(c *gin.Context) {
	var params struct {
		Domains    string `json:"domains" form:"domains"`
		TimeoutSec int    `json:"timeout" form:"timeout"`
	}
	if err := c.ShouldBind(&params); err != nil {
		jsonMsg(c, "", err)
		return
	}

	if params.TimeoutSec <= 0 {
		params.TimeoutSec = 3
	}

	var domains []string
	for _, d := range strings.Split(params.Domains, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}

	scanner := service.GetScannerService()
	results := scanner.ValidateDomains(domains, params.TimeoutSec)
	jsonObj(c, results, nil)
}

func (a *ApiService) PostStopScan(c *gin.Context) {
	scanner := service.GetScannerService()
	scanner.StopScan()
	jsonMsg(c, "success", nil)
}

func (a *ApiService) PostPauseScan(c *gin.Context) {
	scanner := service.GetScannerService()
	scanner.PauseScan()
	jsonMsg(c, "success", nil)
}

func (a *ApiService) PostResumeScan(c *gin.Context) {
	scanner := service.GetScannerService()
	scanner.ResumeScan()
	jsonMsg(c, "success", nil)
}

func (a *ApiService) GetScanStatus(c *gin.Context) {
	scanner := service.GetScannerService()
	status := scanner.GetStatus()
	jsonObj(c, status, nil)
}

func (a *ApiService) GetServerIp(c *gin.Context) {
	scanner := service.GetScannerService()
	ip := scanner.GetServerPublicIP()
	jsonObj(c, ip, nil)
}

func (a *ApiService) PostSaveScannerReality(c *gin.Context) {
	var params struct {
		IP     string `json:"ip" form:"ip"`
		Domain string `json:"domain" form:"domain"`
	}
	if err := c.ShouldBind(&params); err != nil {
		jsonMsg(c, "", err)
		return
	}

	if params.IP == "" || params.Domain == "" {
		jsonMsg(c, "", fmt.Errorf("ip or domain is empty"))
		return
	}

	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err == nil {
			tx.Commit()
			_ = a.ConfigService.RestartCore()
		} else {
			tx.Rollback()
		}
	}()

	// 1. 生成 Reality 密钥对
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	publicKey := privateKey.PublicKey()
	privKeyStr := base64.RawURLEncoding.EncodeToString(privateKey[:])
	pubKeyStr := base64.RawURLEncoding.EncodeToString(publicKey[:])

	// 2. 生成 short_id
	tempUuid, err := uuid.NewV4()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	shortId := strings.ReplaceAll(tempUuid.String(), "-", "")[:16]

	// 3. 构造 Tls 证书项
	serverConfig := fmt.Sprintf(`{"enabled":true,"server_name":"%s","reality":{"enabled":true,"handshake":{"server":"%s","server_port":443},"private_key":"%s","short_id":["%s"]}}`, params.Domain, params.IP, privKeyStr, shortId)
	clientConfig := fmt.Sprintf(`{"reality":{"public_key":"%s"},"utls":{"fingerprint":"chrome"}}`, pubKeyStr)

	tlsModel := model.Tls{
		Name:   "Reality-" + params.Domain,
		Server: json.RawMessage(serverConfig),
		Client: json.RawMessage(clientConfig),
	}
	err = tx.Create(&tlsModel).Error
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	// 4. 生成随机端口与 VLESS Inbound
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(50000) + 10000
	tag := fmt.Sprintf("vless-reality-%d", port)
	inboundOptions := fmt.Sprintf(`{"listen":"::","listen_port":%d,"flow":"xtls-rprx-vision"}`, port)

	inbound := model.Inbound{
		Type:    "vless",
		Tag:     tag,
		TlsId:   tlsModel.Id,
		Options: json.RawMessage(inboundOptions),
	}

	err = tx.Create(&inbound).Error
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	// 5. 新建 VLESS 客户端并关联
	clientUuid, err := uuid.NewV4()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	clientConfigStr := fmt.Sprintf(`{"vless":{"id":"%s","flow":"xtls-rprx-vision"}}`, clientUuid.String())
	clientInbounds := fmt.Sprintf(`[%d]`, inbound.Id)

	client := model.Client{
		Name:     "client-" + tag,
		Enable:   true,
		Inbounds: json.RawMessage(clientInbounds),
		Config:   json.RawMessage(clientConfigStr),
	}

	err = tx.Create(&client).Error
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	c.JSON(200, gin.H{"status": "success", "obj": gin.H{
		"port":   port,
		"tag":    tag,
		"domain": params.Domain,
		"uuid":   clientUuid.String(),
	}})
}
