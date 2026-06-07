package service

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sort"
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
	IsPaused   bool         `json:"is_paused"`
	Total      int          `json:"total"`
	Scanned    int          `json:"scanned"`
	FoundCount int          `json:"found_count"`
	Results    []ScanResult `json:"results"`
}

type ScannerService struct {
	isRunning    bool
	isPaused     bool
	totalIPs     int
	scannedIPs   int
	foundDomains []ScanResult
	stopChan     chan struct{}
	pauseCond    *sync.Cond
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
		globalScanner.pauseCond = sync.NewCond(&globalScanner.mu)
	})
	return globalScanner
}

func (s *ScannerService) GetStatus() ScanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	resultsCopy := make([]ScanResult, len(s.foundDomains))
	copy(resultsCopy, s.foundDomains)

	return ScanStatus{
		IsRunning:  s.isRunning,
		IsPaused:   s.isPaused,
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
		s.isPaused = false
		close(s.stopChan)
		s.pauseCond.Broadcast() // 唤醒所有可能被挂起在暂停状态的 Worker
	}
}

func (s *ScannerService) StartScan(targets string, threads int, timeoutSec int, durationSec int, heuristicSni string) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("扫描任务已在运行中")
	}

	s.isRunning = true
	s.isPaused = false
	s.stopChan = make(chan struct{})
	s.foundDomains = make([]ScanResult, 0)
	s.scannedIPs = 0
	s.totalIPs = 0
	s.mu.Unlock()

	go s.runScanTask(targets, threads, timeoutSec, durationSec, heuristicSni)
	return nil
}

func (s *ScannerService) PauseScan() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning && !s.isPaused {
		s.isPaused = true
	}
}

func (s *ScannerService) ResumeScan() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning && s.isPaused {
		s.isPaused = false
		s.pauseCond.Broadcast() // 唤醒所有 Worker 继续扫描
	}
}

func (s *ScannerService) runScanTask(targets string, threads int, timeoutSec int, durationSec int, heuristicSni string) {
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	ips := parseTargetsToIPs(targets)
	if len(ips) == 0 {
		return
	}

	// 随机抽样限制在 2000 个以保障扫描效率
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

	// 启动 Worker 进行并发扫描
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

					// 执行前检查暂停状态
					s.mu.Lock()
					for s.isPaused && s.isRunning {
						s.pauseCond.Wait()
					}
					if !s.isRunning {
						s.mu.Unlock()
						return
					}
					s.mu.Unlock()

					s.scanIPAddress(ip, timeout, heuristicSni)
					s.mu.Lock()
					s.scannedIPs++
					s.mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
}

func (s *ScannerService) scanIPAddress(ip net.IP, timeout time.Duration, heuristicSni string) {
	hostPort := net.JoinHostPort(ip.String(), "443")
	dialer := &net.Dialer{Timeout: timeout}

	// 1. 尝试第一次常规握手（不带 SNI 探测）
	var state tls.ConnectionState
	var hasCerts bool

	conn, err := dialer.Dial("tcp", hostPort)
	if err == nil {
		tlsConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
			MaxVersion:         tls.VersionTLS13,
		})
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
		err = tlsConn.Handshake()
		if err == nil {
			state = tlsConn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				hasCerts = true
			}
		}
		tlsConn.Close()
		conn.Close()
	}

	// 2. 如果常规握手失败，且配置了启发式 SNI，则启用 SNI 回退探测
	if (!hasCerts || err != nil) && heuristicSni != "" {
		conn, err = dialer.Dial("tcp", hostPort)
		if err == nil {
			tlsConn := tls.Client(conn, &tls.Config{
				ServerName:         heuristicSni,
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
				MaxVersion:         tls.VersionTLS13,
			})
			_ = tlsConn.SetDeadline(time.Now().Add(timeout))
			err = tlsConn.Handshake()
			if err == nil {
				state = tlsConn.ConnectionState()
				if len(state.PeerCertificates) > 0 {
					hasCerts = true
				}
			}
			tlsConn.Close()
			conn.Close()
		}
	}

	if !hasCerts {
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

	// 3. 对解析出的域名分别进行 Reality TLS 1.3/X25519 连通性测试
	for domain := range candidates {
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

	// 强制要求 TLS 1.3 且带上 X25519 的 Reality 最优配置进行协商
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

	blacklistedTerms := []string{
		"self-signed", "localhost", "kubernetes", "ingress", "traefik",
		"minica", "mkcert", "default", "root ca", "node", "router",
	}
	for _, term := range blacklistedTerms {
		if strings.Contains(issuer, term) {
			return ScanResult{}, fmt.Errorf("untrusted self-signed/internal issuer")
		}
	}

	// 模拟浏览器发送极简 HTTP GET 请求以验证网页状态与后台关键字
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

	if statusCode == 301 || statusCode == 302 || statusCode == 307 || statusCode == 308 {
		return ScanResult{}, fmt.Errorf("redirect occurred")
	}

	bodyStartIndex := strings.Index(respStr, "\r\n\r\n")
	bodyContent := ""
	if bodyStartIndex != -1 && len(respStr) > bodyStartIndex+4 {
		bodyContent = respStr[bodyStartIndex+4:]
	}

	if len(bodyContent) < 100 {
		if len(respStr) < 200 {
			return ScanResult{}, fmt.Errorf("response size too small")
		}
	}

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

// 模式 A：大厂知名域名一键测速校验模式的后台逻辑实现
func (s *ScannerService) ValidateDomains(domains []string, timeoutSec int) []ScanResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]ScanResult, 0)
	timeout := time.Duration(timeoutSec) * time.Second

	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			
			// 1. 进行 DNS 解析，获取真实 IP
			ips, err := net.LookupIP(domain)
			if err != nil || len(ips) == 0 {
				return
			}
			
			// 选择解析出的第一个 IPv4 进行测试
			var targetIP net.IP
			for _, ip := range ips {
				if ip.To4() != nil {
					targetIP = ip
					break
				}
			}
			if targetIP == nil {
				targetIP = ips[0]
			}

			// 2. 调用核心测速与证书/网页校验方法
			res, err := s.testTargetDomain(targetIP, domain, timeout)
			if err == nil {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}(d)
	}

	wg.Wait()

	// 按照物理延迟从低到高进行完美排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Delay < results[j].Delay
	})

	return results
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
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return strings.TrimSpace(string(body))
		}
	}
	resp, err = client.Get("http://icanhazip.com")
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return strings.TrimSpace(string(body))
		}
	}

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

// 优化版的 IP 解析与生成，避免内存暴涨
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

			// 1. 处理 CIDR 段，最多只允许生成前 2048 个 IP 限制内存
			if _, ipNet, err := net.ParseCIDR(part); err == nil {
				count := 0
				for ip := cloneIP(ipNet.IP); ipNet.Contains(ip) && count < 2048; incrementIP(ip) {
					ips = append(ips, cloneIP(ip))
					count++
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
						count := 0
						for ip := cloneIP(startIP); bytesLessThanOrEqual(ip, endIP) && count < 2048; incrementIP(ip) {
							ips = append(ips, cloneIP(ip))
							count++
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
