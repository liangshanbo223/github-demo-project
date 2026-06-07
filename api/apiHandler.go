package api

import (
	"strconv"
	"strings"

	"github.com/liangshanbo223/github-demo-project/util/common"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	ApiService
	apiv2 *APIv2Handler
}

func NewAPIHandler(g *gin.RouterGroup, a2 *APIv2Handler) {
	a := &APIHandler{
		apiv2: a2,
	}
	a.initRouter(g)
}

func (a *APIHandler) initRouter(g *gin.RouterGroup) {
	g.GET("/node/config", a.nodeConfigHandler)
	g.POST("/node/report", a.nodeReportHandler)

	g.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasSuffix(path, "login") && !strings.HasSuffix(path, "logout") && !strings.HasSuffix(path, "recovery") {
			checkLogin(c)
		}
	})
	g.POST("/scanner/validate", a.validateDomainsHandler)
	g.POST("/:postAction", a.postHandler)
	g.GET("/:getAction", a.getHandler)
}

func (a *APIHandler) validateDomainsHandler(c *gin.Context) {
	a.ApiService.PostValidateDomains(c)
}

func (a *APIHandler) nodeConfigHandler(c *gin.Context) {
	nodeIdStr := c.Query("node_id")
	token := c.Query("token")
	if nodeIdStr == "" || token == "" {
		c.JSON(400, gin.H{"status": "failed", "error": "missing node_id or token"})
		return
	}
	nodeId, err := strconv.Atoi(nodeIdStr)
	if err != nil {
		c.JSON(400, gin.H{"status": "failed", "error": "invalid node_id"})
		return
	}
	a.ApiService.NodeConfig(c, uint(nodeId), token)
}

func (a *APIHandler) nodeReportHandler(c *gin.Context) {
	a.ApiService.NodeReport(c)
}

func (a *APIHandler) postHandler(c *gin.Context) {
	loginUser := GetLoginUser(c)
	action := c.Param("postAction")

	switch action {
	case "login":
		a.ApiService.Login(c)
	case "changePass":
		a.ApiService.ChangePass(c)
	case "save":
		a.ApiService.Save(c, loginUser)
	case "restartApp":
		a.ApiService.RestartApp(c)
	case "restartSb":
		a.ApiService.RestartSb(c)
	case "linkConvert":
		a.ApiService.LinkConvert(c)
	case "subConvert":
		a.ApiService.SubConvert(c)
	case "importdb":
		a.ApiService.ImportDb(c)
	case "addToken":
		a.ApiService.AddToken(c)
		a.apiv2.ReloadTokens()
	case "deleteToken":
		a.ApiService.DeleteToken(c)
		a.apiv2.ReloadTokens()
	case "acmeIssue":
		a.ApiService.PostAcmeIssue(c)
	case "updateSystem":
		a.ApiService.UpdateSystem(c)
	case "rollbackSystem":
		a.ApiService.RollbackSystem(c)
	case "recovery":
		a.ApiService.RecoveryPassword(c)
	case "diagnose":
		a.ApiService.PostDiagnose(c)
	case "startScan":
		a.ApiService.PostStartScan(c)
	case "stopScan":
		a.ApiService.PostStopScan(c)
	case "saveScannerReality":
		a.ApiService.PostSaveScannerReality(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}

func (a *APIHandler) getHandler(c *gin.Context) {
	action := c.Param("getAction")

	switch action {
	case "logout":
		a.ApiService.Logout(c)
	case "load":
		a.ApiService.LoadData(c)
	case "inbounds", "outbounds", "endpoints", "services", "tls", "clients", "config":
		err := a.ApiService.LoadPartialData(c, []string{action})
		if err != nil {
			jsonMsg(c, action, err)
		}
		return
	case "users":
		a.ApiService.GetUsers(c)
	case "settings":
		a.ApiService.GetSettings(c)
	case "stats":
		a.ApiService.GetStats(c)
	case "status":
		a.ApiService.GetStatus(c)
	case "onlines":
		a.ApiService.GetOnlines(c)
	case "logs":
		a.ApiService.GetLogs(c)
	case "changes":
		a.ApiService.CheckChanges(c)
	case "keypairs":
		a.ApiService.GetKeypairs(c)
	case "getdb":
		a.ApiService.GetDb(c)
	case "tokens":
		a.ApiService.GetTokens(c)
	case "singbox-config":
		a.ApiService.GetSingboxConfig(c)
	case "checkOutbound":
		a.ApiService.GetCheckOutbound(c)
	case "acmeLog":
		a.ApiService.GetAcmeLogStream(c)
	case "backups":
		a.ApiService.GetBackups(c)
	case "sysLog":
		a.ApiService.GetSysLogStream(c)
	case "scanStatus":
		a.ApiService.GetScanStatus(c)
	case "serverIp":
		a.ApiService.GetServerIp(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}
