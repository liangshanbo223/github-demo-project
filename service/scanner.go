package service

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ScanResult struct {
	IP         string `json:"ip"`
	Domain     string `json:"domain"`
	ALPN       string `json:"alpn"`
	TLSVersion string `json:"tls_version"`
	Delay      int64  `json:"delay"`
	StatusCode int    `json:"status_code"`
}

type ScanStatus struct {
	IsRunning  bool         `json:"is_running"`
	Total      int          `json:"total"`
	Scanned    int          `json:"scanned"`
	FoundCount int          `json:"found_count"`
	Results    []ScanResult `json:"results"`
}

type ScannerService struct {
	isRunning    bool
	totalIPs     int
	scannedIPs   int
	foundDomains []ScanResult
	stopChan     chan struct{}
	mu           sync.Mutex
}

var (
	globalScanner *ScannerService
	scannerOnce   sync.Once
)

func GetScannerService() *ScannerService {
	scannerOnce.Do(func() {
		globalScanner = &ScannerService{
			foundDomains: make([]ScanResult, 0),
		}
	})
	return globalScanner
}

func (s *ScannerService) GetStatus() ScanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Copy results to prevent concurrent map read/write
	resultsCopy := make([]ScanResult, len(s.foundDomains))
	copy(resultsCopy, s.foundDomains)

	return ScanStatus{
		IsRunning:  s.isRunning,
		Total:      s.totalIPs,
		Scanned:    s.scannedIPs,
		FoundCount: len(s.foundDomains),
		Results:    resultsCopy,
	}
}

func (s *ScannerService) StopScan() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning {
		s.isRunning = false
		close(s.stopChan)
	}
}

func (s *ScannerService) StartScan(targets string, threads int, timeoutSec int, durationSec int) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("扫描任务已在运行中")
	}

	s.isRunning = true
	s.stopChan = make(chan struct{})
	s.foundDomains = make([]ScanResult, 0)
	s.scannedIPs = 0
	s.totalIPs = 0
	s.mu.Unlock()

	go s.runScanTask(targets, threads, timeoutSec, durationSec)
	return nil
}

func (s *ScannerService) runScanTask(targets string, threads int, timeoutSec int, durationSec int) {
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	ips := parseTargetsToIPs(targets)
	if len(ips) == 0 {
		return
	}

	// 随机抽样限制在 2000 个以保障 5 分钟内扫完
	if len(ips) > 2000 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(ips), func(i, j int) {
			ips[i], ips[j] = ips[j], ips[i]
		})
		ips = ips[:2000]
	}

	s.mu.Lock()
	s.totalIPs = len(ips)
	s.mu.Unlock()

	ipChan := make(chan net.IP, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	var wg sync.WaitGroup
	timeout := time.Duration(timeoutSec) * time.Second
	limitTimer := time.NewTimer(time.Duration(durationSec) * time.Second)
	defer limitTimer.Stop()

	// 启动 Worker
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-s.stopChan:
					return
				case <-limitTimer.C:
					return
				case ip, ok := <-ipChan:
					if !ok {
						return
					}
					s.scanIPAddress(ip, timeout)
					s.mu.Lock()
					s.scannedIPs++
					s.mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
}

func (s *ScannerService) scanIPAddress(ip net.IP, timeout time.Duration) {
	hostPort := net.JoinHostPort(ip.String(), "443")
	
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", hostPort)
	if err != nil {
		return
	}
	defer conn.Close()

	// 第一次握手：不带 SNI 获取证书域名列表
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS13,
	})
	
	_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	err = tlsConn.Handshake()
	if err != nil {
		return
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return
	}

	cert := state.PeerCertificates[0]
	candidates := make(map[string]bool)
	if cert.Subject.CommonName != "" {
		candidates[cert.Subject.CommonName] = true
	}
	for _, name := range cert.DNSNames {
		candidates[name] = true
	}

	// 针对获取的每个候选域名作 Reality 兼容性判定与正常网站强检测
	for domain := range candidates {
		// 转换通配符 *.example.com => www.example.com
		if strings.Contains(domain, "*") {
			domain = strings.Replace(domain, "*.", "www.", 1)
		}
		domain = strings.TrimSpace(domain)
		if domain == "" || strings.HasSuffix(domain, ".local") || strings.HasSuffix(domain, ".lan") {
			continue
		}

		res, err := s.testTargetDomain(ip, domain, timeout)
		if err == nil {
			s.mu.Lock()
			s.foundDomains = append(s.foundDomains, res)
			s.mu.Unlock()
		}
	}
}

func (s *ScannerService) testTargetDomain(ip net.IP, domain string, timeout time.Duration) (ScanResult, error) {
	hostPort := net.JoinHostPort(ip.String(), "443")
	
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", hostPort)
	if err != nil {
		return ScanResult{}, err
	}
	defer conn.Close()

	// 强制要求 TLS 1.3 且支持 X25519
	tlsCfg := &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		CurvePreferences:   []tls.CurveID{tls.X25519},
		NextProtos:         []string{"h2", "http/1.1"},
	}

	tlsConn := tls.Client(conn, tlsCfg)
	_ = tlsConn.SetDeadline(time.Now().Add(timeout))

	startTime := time.Now()
	err = tlsConn.Handshake()
	if err != nil {
		return ScanResult{}, err
	}
	delay := time.Since(startTime).Milliseconds()

	state := tlsConn.ConnectionState()
	if state.Version != tls.VersionTLS13 {
		return ScanResult{}, fmt.Errorf("not tls 1.3")
	}

	alpn := state.NegotiatedProtocol
	if alpn == "" {
		alpn = "http/1.1"
	}

	// 1. 验证证书是否合法（排除无效自签与内网自建 CA）
	if len(state.PeerCertificates) == 0 {
		return ScanResult{}, fmt.Errorf("no peer certs")
	}
	cert := state.PeerCertificates[0]

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return ScanResult{}, fmt.Errorf("cert expired or not active")
	}

	issuerOrg := ""
	if len(cert.Issuer.Organization) > 0 {
		issuerOrg = cert.Issuer.Organization[0]
	}
	issuer := strings.ToLower(cert.Issuer.CommonName + issuerOrg)

	// 黑名单自建 Issuer，避免不安全的内网服务
	blacklistedTerms := []string{
		"self-signed", "localhost", "kubernetes", "ingress", "traefik", 
		"minica", "mkcert", "default", "root ca", "node", "router",
	}
	for _, term := range blacklistedTerms {
		if strings.Contains(issuer, term) {
			return ScanResult{}, fmt.Errorf("untrusted self-signed/internal issuer")
		}
	}

	// 2. 发起 HTTP 头部和页面抓取以确定为正常网站 (排除空响应或路由器/网关后台等)
	reqStr := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\nConnection: close\r\n\r\n", domain)
	_, err = tlsConn.Write([]byte(reqStr))
	if err != nil {
		return ScanResult{}, err
	}

	buf := make([]byte, 2048)
	n, err := tlsConn.Read(buf)
	if err != nil && n == 0 {
		return ScanResult{}, err
	}

	respStr := string(buf[:n])
	
	// 解析状态码
	statusCode := 200
	lines := strings.Split(respStr, "\r\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "HTTP/") {
		parts := strings.Split(lines[0], " ")
		if len(parts) > 1 {
			if code, err := strconv.Atoi(parts[1]); err == nil {
				statusCode = code
			}
		}
	}

	// 丢弃重定向节点
	if statusCode == 301 || statusCode == 302 || statusCode == 307 || statusCode == 308 {
		return ScanResult{}, fmt.Errorf("redirect occurred")
	}

	// 解析 Body 长度与内容
	bodyStartIndex := strings.Index(respStr, "\r\n\r\n")
	bodyContent := ""
	if bodyStartIndex != -1 && len(respStr) > bodyStartIndex+4 {
		bodyContent = respStr[bodyStartIndex+4:]
	}

	// 判定网页的字节长度 (排除空网页)
	if len(bodyContent) < 100 {
		// 某些大厂网站根路径直接返回 403 或者是 400，但头信息可能长于 100，这也是正常
		if len(respStr) < 200 {
			return ScanResult{}, fmt.Errorf("response size too small")
		}
	}

	// 对 HTML 页面标题进行识别，过滤 NAS 路由器、设备网关
	bodyLower := strings.ToLower(bodyContent)
	title := extractHTMLTitle(bodyLower)
	gatewayKeywords := []string{
		"synology", "nas", "router", "openwrt", "ikuai", "pfsense",
		"panabit", "login", "h3c", "tp-link", "netgear", "d-link",
		"管理系统", "后台登录", "网关", "监控", "宝塔", "aapanel",
	}
	for _, term := range gatewayKeywords {
		if strings.Contains(title, term) {
			return ScanResult{}, fmt.Errorf("gateway/admin portal detected")
		}
	}

	return ScanResult{
		IP:         ip.String(),
		Domain:     domain,
		ALPN:       alpn,
		TLSVersion: "TLS 1.3",
		Delay:      delay,
		StatusCode: statusCode,
	}, nil
}

func extractHTMLTitle(body string) string {
	start := strings.Index(body, "<title>")
	if start == -1 {
		return ""
	}
	end := strings.Index(body, "</title>")
	if end == -1 || end <= start+7 {
		return ""
	}
	return body[start+7 : end]
}

func (s *ScannerService) GetServerPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return strings.TrimSpace(string(body))
		}
	}
	// 备用
	resp, err = client.Get("http://icanhazip.com")
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return strings.TrimSpace(string(body))
		}
	}
	
	// 若无公网连接，获取网卡 IP
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

// 辅助解析 IP 生成器
func parseTargetsToIPs(targets string) []net.IP {
	var ips []net.IP
	targets = strings.ReplaceAll(targets, "\r", "")
	lines := strings.Split(targets, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			
			// 1. 处理 CIDR
			if _, ipNet, err := net.ParseCIDR(part); err == nil {
				// 获取首个 IP，逐个累加
				for ip := cloneIP(ipNet.IP); ipNet.Contains(ip); incrementIP(ip) {
					ips = append(ips, cloneIP(ip))
				}
				continue
			}

			// 2. 处理 IP 范围 1.1.1.1-1.1.1.10
			if strings.Contains(part, "-") {
				subParts := strings.Split(part, "-")
				if len(subParts) == 2 {
					startIP := net.ParseIP(strings.TrimSpace(subParts[0]))
					endIP := net.ParseIP(strings.TrimSpace(subParts[1]))
					if startIP != nil && endIP != nil {
						for ip := cloneIP(startIP); bytesLessThanOrEqual(ip, endIP); incrementIP(ip) {
							ips = append(ips, cloneIP(ip))
						}
					}
				}
				continue
			}

			// 3. 处理单 IP
			if ip := net.ParseIP(part); ip != nil {
				ips = append(ips, ip)
				continue
			}
		}
	}
	return ips
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func cloneIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func bytesLessThanOrEqual(ip1, ip2 net.IP) bool {
	for i := 0; i < len(ip1); i++ {
		if ip1[i] < ip2[i] {
			return true
		} else if ip1[i] > ip2[i] {
			return false
		}
	}
	return true
}
