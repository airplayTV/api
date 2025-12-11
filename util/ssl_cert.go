package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	rand2 "math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var email = "localhost@airplayTV.org" // 标识证书用

func MakeDomainCertificate(certificateName string, dnsNames []string) (cert, key string, err error) {
	_ = MkdirAll(filepath.Join(AppPath(), "www/certs/1.txt"))
	cert = filepath.Join(AppPath(), fmt.Sprintf("www/certs/%s.crt", certificateName)) // 同 ca.crt 文件 // 同 cert.pem 文件
	key = filepath.Join(AppPath(), fmt.Sprintf("www/certs/%s.key", certificateName))  // 同 ca.key 文件 // 同 key.pem 文件
	//_, certErr := os.Stat(cert)
	//_, keyErr := os.Stat(key)
	//if certErr == nil && keyErr == nil && !debug { // debug模式不缓存
	//	log.Println("[证书文件已存在]", cert)
	//	return
	//}
	var sn = big.NewInt(int64(time.Now().Year()*10000000000000 + rand2.IntN(99999999)))
	log.Println("[开始生成证书]", sn)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return //生成RSA密钥出错
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: sn, // 序列号
		Subject: pkix.Name{
			CommonName:   certificateName,
			Organization: []string{certificateName},
			Country:      []string{"USA"},
			Province:     []string{"USA"},
		},
		Issuer: pkix.Name{
			CommonName:   certificateName,
			Organization: []string{certificateName},
			Country:      []string{"USA"},
			Province:     []string{"USA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 1年有效期
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true, // 标记为CA证书
		DNSNames:              dnsNames,
		EmailAddresses:        []string{email},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return //创建自签名证书失败
	}

	// 将证书写入PEM文件
	certOut, err := os.Create(cert)
	if err != nil {
		return //创建证书文件失败
	}
	defer func() { _ = certOut.Close() }()
	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return //写入证书失败
	}

	// 将私钥写入PEM文件
	keyOut, err := os.Create(key)
	if err != nil {
		return //创建私钥文件失败
	}
	defer func() { _ = keyOut.Close() }()
	privateBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return //序列化私钥失败
	}
	if err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}); err != nil {
		return //写入私钥失败
	}

	return
}

func AppendCertsFromPEM(pemCerts []byte) bool {
	// 信任证书
	return x509.NewCertPool().AppendCertsFromPEM(pemCerts)
}

func AddCertToRoot(crt string) ([]byte, error) {
	cmd := exec.Command("certutil", "-addstore", "root", crt)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	output, _ = GBKToUTF8(output)
	return output, nil
}

func ReplaceCertToRoot(crt string, debug bool) ([]byte, error) {
	output, err := exec.Command("certutil", "-delstore", "root", email).CombinedOutput()
	if err != nil {
		log.Println("[移除旧证书失败]", err.Error())
	}
	if debug {
		output, _ = GBKToUTF8(output)
		log.Println("[移除旧证书]", strings.TrimSpace(string(output)))
	}

	time.Sleep(time.Second / 4)

	cmd := exec.Command("certutil", "-addstore", "root", crt)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	output, _ = GBKToUTF8(output)
	if debug {
		log.Println("[添加新证书]", strings.TrimSpace(string(output)))
	}
	return output, nil
}
