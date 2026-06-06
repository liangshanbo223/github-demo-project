package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/admin8800/s-ui/core"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func RunNode() {
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	var apiAddr string
	var nodeId int
	var token string
	var syncInterval int

	nodeCmd.StringVar(&apiAddr, "api", "", "主控 API 地址 (例如 http://127.0.0.1:2053/api)")
	nodeCmd.IntVar(&nodeId, "node", 0, "节点 ID")
	nodeCmd.StringVar(&token, "token", "", "节点 Token")
	nodeCmd.IntVar(&syncInterval, "interval", 30, "配置同步与流量上报间隔(秒)")

	if len(os.Args) < 3 {
		fmt.Println("用法: sui node [options]")
		nodeCmd.PrintDefaults()
		return
	}

	err := nodeCmd.Parse(os.Args[2:])
	if err != nil {
		fmt.Println("解析命令行参数错误:", err)
		return
	}

	if apiAddr == "" || nodeId == 0 || token == "" {
		fmt.Println("错误: --api, --node, 以及 --token 参数为必填项！")
		nodeCmd.PrintDefaults()
		return
	}

	// Ensure apiAddr does not end with /
	if apiAddr[len(apiAddr)-1] == '/' {
		apiAddr = apiAddr[:len(apiAddr)-1]
	}

	fmt.Printf("正在启动 s-ui 节点 (ID: %d)...\n", nodeId)
	fmt.Printf("主控地址: %s\n", apiAddr)
	fmt.Printf("同步间隔: %d 秒\n", syncInterval)

	// 初始化 core
	nodeCore := core.NewCore()

	// 同步定时器
	ticker := time.NewTicker(time.Duration(syncInterval) * time.Second)
	defer ticker.Stop()

	// 捕获系统退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 首次运行同步
	syncNode(apiAddr, nodeId, token, nodeCore)

	for {
		select {
		case <-sigCh:
			fmt.Println("正在关闭节点...")
			nodeCore.Stop()
			return
		case <-ticker.C:
			syncNode(apiAddr, nodeId, token, nodeCore)
		}
	}
}

var lastConfigHash []byte

func syncNode(apiAddr string, nodeId int, token string, nodeCore *core.Core) {
	// 1. 获取最新配置
	url := fmt.Sprintf("%s/node/config?node_id=%d&token=%s", apiAddr, nodeId, token)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("[%s] 同步配置网络错误: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[%s] 同步配置主控返回错误: HTTP %d - %s\n", time.Now().Format("2006-01-02 15:04:05"), resp.StatusCode, string(body))
		return
	}

	configBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[%s] 读取配置内容失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}

	// 比较哈希，如有变更重新加载
	if lastConfigHash == nil || !bytes.Equal(lastConfigHash, configBytes) {
		fmt.Printf("[%s] 发现配置更新，正在重新加载内核...\n", time.Now().Format("2006-01-02 15:04:05"))
		if nodeCore.IsRunning() {
			err = nodeCore.Stop()
			if err != nil {
				fmt.Printf("[%s] 停止旧内核失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			}
		}

		err = nodeCore.Start(configBytes)
		if err != nil {
			fmt.Printf("[%s] 启动新内核失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			return
		}

		lastConfigHash = configBytes
		fmt.Printf("[%s] 内核重载成功。\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	// 2. 收集系统指标与流量数据
	var cpuPercent float64
	percents, err := cpu.Percent(0, false)
	if err == nil && len(percents) > 0 {
		cpuPercent = percents[0]
	}

	var memPercent float64
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		memPercent = memInfo.UsedPercent
	}

	type TrafficItem struct {
		Tag  string `json:"tag"`
		Up   int64  `json:"up"`
		Down int64  `json:"down"`
	}

	var trafficList []TrafficItem

	if nodeCore.IsRunning() {
		box := nodeCore.GetInstance()
		if box != nil {
			st := box.StatsTracker()
			if st != nil {
				stats := st.GetStats()
				if stats != nil {
					userStats := make(map[string]*TrafficItem)
					for _, stat := range *stats {
						if stat.Resource == "user" {
							item, exists := userStats[stat.Tag]
							if !exists {
								item = &TrafficItem{Tag: stat.Tag}
								userStats[stat.Tag] = item
							}
							if stat.Direction {
								item.Up += stat.Traffic
							} else {
								item.Down += stat.Traffic
							}
						}
					}
					for _, item := range userStats {
						if item.Up > 0 || item.Down > 0 {
							trafficList = append(trafficList, *item)
						}
					}
				}
			}
		}
	}

	// 3. 上报状态给主控
	reportURL := fmt.Sprintf("%s/node/report", apiAddr)
	reportData := map[string]interface{}{
		"node_id": nodeId,
		"token":   token,
		"status": map[string]interface{}{
			"cpu": cpuPercent,
			"mem": memPercent,
		},
		"traffic": trafficList,
	}

	reportJSON, err := json.Marshal(reportData)
	if err != nil {
		fmt.Printf("[%s] 序列化上报数据失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}

	req, err := http.NewRequest("POST", reportURL, bytes.NewBuffer(reportJSON))
	if err != nil {
		fmt.Printf("[%s] 创建上报请求失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	reportResp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[%s] 上报数据失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}
	defer reportResp.Body.Close()

	if reportResp.StatusCode != 200 {
		body, _ := io.ReadAll(reportResp.Body)
		fmt.Printf("[%s] 上报响应错误: HTTP %d - %s\n", time.Now().Format("2006-01-02 15:04:05"), reportResp.StatusCode, string(body))
		return
	}
}
