package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// GenerateCACert 生成根证书和根私钥，保存到指定路径
func GenerateCACert(certPath, keyPath string) error {
	// 1. 生成 RSA 2048 私钥
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// 2. 构造自签名 CA 模板
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"S-UI Custom Self-Signed Root CA"},
			CommonName:   "S-UI Custom Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10年有效期
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	// 3. 创建自签名证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// 4. 确保目录存在
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 5. 写入 PEM 证书
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	// 6. 写入 PEM 私钥
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}

// GenerateServerCertByCA 根据已有的 CA 证书和私钥，为特定的域名（或IP）派生签发服务器证书对
func GenerateServerCertByCA(caCertPath, caKeyPath, commonName string) (certPEM []byte, keyPEM []byte, err error) {
	// 1. 读取并解析 CA 证书
	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, err
	}
	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		return nil, nil, errors.New("failed to parse CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	// 2. 读取并解析 CA 私钥
	caKeyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("failed to parse CA private key PEM")
	}
	caPriv, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	// 3. 生成服务器 RSA 私钥
	serverPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// 4. 构造服务器证书模板
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"S-UI Custom Server Node"},
			CommonName:   commonName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // 1年有效期
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}

	// 5. 设置 SAN (Subject Alternative Names)
	if ip := net.ParseIP(commonName); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{commonName}
	}

	// 6. 用 CA 签名服务器证书
	serverDerBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverPriv.PublicKey, caPriv)
	if err != nil {
		return nil, nil, err
	}

	// 7. 编码为 PEM 格式的字节
	certBlock := &pem.Block{Type: "CERTIFICATE", Bytes: serverDerBytes}
	certPEM = pem.EncodeToMemory(certBlock)

	serverPrivBytes, err := x509.MarshalPKCS8PrivateKey(serverPriv)
	if err != nil {
		return nil, nil, err
	}
	keyBlockOut := &pem.Block{Type: "PRIVATE KEY", Bytes: serverPrivBytes}
	keyPEM = pem.EncodeToMemory(keyBlockOut)

	return certPEM, keyPEM, nil
}
