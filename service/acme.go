package service

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liangshanbo223/github-demo-project/config"
	"github.com/liangshanbo223/github-demo-project/logger"
)

var (
	acmeLogCh   = make(chan string, 100)
	acmeLogMu   sync.Mutex
	acmeRunning bool
)

type AcmeService struct{}

// GetAcmeLogChan 获取 ACME 实时日志通道
func (s *AcmeService) GetAcmeLogChan() chan string {
	return acmeLogCh
}

// CheckAcmeRunning 检查 ACME 任务是否正在运行
func (s *AcmeService) CheckAcmeRunning() bool {
	acmeLogMu.Lock()
	defer acmeLogMu.Unlock()
	return acmeRunning
}

// CheckPortOccupied 检测端口是否被占用，返回是否占用和冲突进程信息
func (s *AcmeService) CheckPortOccupied(port int) (bool, string) {
	address := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		// 端口被占用，尝试获取进程名字 (在 Linux 下可以通过读取 /proc 或者执行 lsof/fuser 探测)
		// 极简实现：直接尝试运行 fuser 或是 lsof 辅助获取进程
		processName := "未知进程"
		cmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -i :%d -t | xargs ps -o comm= 2>/dev/null", port))
		out, errCmd := cmd.Output()
		if errCmd == nil && len(out) > 0 {
			processName = strings.TrimSpace(string(out))
		} else {
			cmd2 := exec.Command("sh", "-c", fmt.Sprintf("fuser %d/tcp 2>/dev/null | xargs ps -o comm= 2>/dev/null", port))
			out2, errCmd2 := cmd2.Output()
			if errCmd2 == nil && len(out2) > 0 {
				processName = strings.TrimSpace(string(out2))
			}
		}
		return true, processName
	}
	l.Close()
	return false, ""
}

// WriteLog 往通道写入日志
func (s *AcmeService) writeLog(msg string) {
	select {
	case acmeLogCh <- msg:
	default:
		// 通道满，丢弃旧日志
	}
}

// InstallAcmeIfNotExist 自动安装 acme.sh
func (s *AcmeService) InstallAcmeIfNotExist() error {
	homeDir, _ := os.UserHomeDir()
	acmePath := filepath.Join(homeDir, ".acme.sh/acme.sh")
	if _, err := os.Stat(acmePath); err == nil {
		return nil
	}

	s.writeLog("[ACME] 正在下载并安装 acme.sh 脚本，请稍候...")
	cmd := exec.Command("sh", "-c", "curl https://get.acme.sh | sh")
	err := cmd.Run()
	if err != nil {
		s.writeLog("[ACME] 安装 acme.sh 失败：" + err.Error())
		return err
	}
	s.writeLog("[ACME] acme.sh 安装成功！")
	return nil
}

// RunAcmeIssue 执行 acme.sh 签发
func (s *AcmeService) RunAcmeIssue(email string, domains []string, dnsProvider string, dnsEnv []string) {
	acmeLogMu.Lock()
	if acmeRunning {
		acmeLogMu.Unlock()
		s.writeLog("[ACME] 已有证书签发任务在运行中，请勿重复发起！")
		return
	}
	acmeRunning = true
	acmeLogMu.Unlock()

	defer func() {
		acmeLogMu.Lock()
		acmeRunning = false
		acmeLogMu.Unlock()
	}()

	// 清理旧日志通道
	for len(acmeLogCh) > 0 {
		<-acmeLogCh
	}

	// 1. 自动检查并安装 acme.sh
	if err := s.InstallAcmeIfNotExist(); err != nil {
		return
	}

	homeDir, _ := os.UserHomeDir()
	acmeSh := filepath.Join(homeDir, ".acme.sh/acme.sh")

	// 2. 准备指令
	var args []string
	args = append(args, "--issue")

	for _, domain := range domains {
		args = append(args, "-d", domain)
	}

	if dnsProvider != "" {
		// 使用 DNS-01 验证
		s.writeLog("[ACME] 采用 DNS 验证模式进行签发 (服务商: " + dnsProvider + ")")
		args = append(args, "--dns", dnsProvider)
	} else {
		// 使用 HTTP-01 免停机内嵌式验证
		s.writeLog("[ACME] 采用零冲突 HTTP-01 面板免停机验证模式...")
		webroot := "/tmp"
		webrootLoc := filepath.Join(webroot, ".well-known/acme-challenge")
		os.MkdirAll(webrootLoc, 0755)
		args = append(args, "-w", webroot)
	}

	if email != "" {
		args = append(args, "--email", email)
	} else {
		args = append(args, "--register-account", "-m", "admin@s-ui.local")
	}

	// 默认使用 Let's Encrypt
	args = append(args, "--server", "letsencrypt")

	s.writeLog(fmt.Sprintf("[ACME] 执行命令: %s %s", acmeSh, strings.Join(args, " ")))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, acmeSh, args...)
	cmd.Dir = homeDir

	// 注入环境变量（用于 DNS API，例如 CF_Key, CF_Email 等）
	if len(dnsEnv) > 0 {
		cmd.Env = append(os.Environ(), dnsEnv...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.writeLog("[ACME] 无法创建 StdoutPipe: " + err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		s.writeLog("[ACME] 启动进程失败: " + err.Error())
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		s.writeLog(scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		s.writeLog("[ACME] 证书申请执行失败，请检查上面日志。")
		logger.Error("ACME issue failed:", err)
		return
	}

	s.writeLog("[ACME] 证书申请成功！开始安装部署...")

	// 3. 安装部署证书到 s-ui 指定目录
	destDir := filepath.Join(config.GetDBFolderPath(), "acme")
	os.MkdirAll(destDir, 0755)
	certDest := filepath.Join(destDir, domains[0]+".crt")
	keyDest := filepath.Join(destDir, domains[0]+".key")

	installArgs := []string{
		"--install-cert", "-d", domains[0],
		"--key-file", keyDest,
		"--fullchain-file", certDest,
	}

	installCmd := exec.Command(acmeSh, installArgs...)
	installCmd.Dir = homeDir
	out, errInstall := installCmd.CombinedOutput()
	if errInstall != nil {
		s.writeLog("[ACME] 证书复制部署失败: " + string(out))
		return
	}

	s.writeLog("[ACME] 证书已成功安装部署至:")
	s.writeLog("   [证书/完整链]: " + certDest)
	s.writeLog("   [私钥文件]: " + keyDest)
	s.writeLog("[ACME] 全流程已顺利完成！")
}

// RenewAllCertificates 每天自动定时续期所有 acme 证书
func (s *AcmeService) RenewAllCertificates() {
	homeDir, _ := os.UserHomeDir()
	acmeSh := filepath.Join(homeDir, ".acme.sh/acme.sh")
	if _, err := os.Stat(acmeSh); err != nil {
		return
	}

	logger.Info("[ACME] 自动检测续期任务启动...")
	cmd := exec.Command(acmeSh, "--cron", "--home", filepath.Join(homeDir, ".acme.sh"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warning("[ACME] 续签检测任务失败:", string(out))
	} else {
		logger.Info("[ACME] 续签检测任务完成:", string(out))
	}
}
