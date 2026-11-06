package node

import (
	"log"
	"os"
	"testing"

	"github.com/InazumaV/V2bX/conf"
)

var l *Lego

func init() {
	// 这两个用例要连 Let's Encrypt 和 Cloudflare，离线/CI 里必然失败。
	// 设置 V2BX_SKIP_NET_TESTS=1 时直接跳过初始化（l 保持 nil，用例内部会 Skip），
	// 这样同包的其它单测（比如 compareUserList）就不会被拖死。
	if os.Getenv("V2BX_SKIP_NET_TESTS") != "" {
		return
	}
	var err error
	l, err = NewLego(&conf.CertConfig{
		CertMode:   "dns",
		Email:      "test@test.com",
		CertDomain: "test.test.com",
		Provider:   "cloudflare",
		DNSEnv: map[string]string{
			"CF_DNS_API_TOKEN": "123",
		},
		CertFile: "./cert/1.pem",
		KeyFile:  "./cert/1.key",
	})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func TestLego_CreateCertByDns(t *testing.T) {
	if l == nil {
		t.Skip("V2BX_SKIP_NET_TESTS: 跳过联网证书测试")
	}
	err := l.CreateCert()
	if err != nil {
		t.Error(err)
	}
}

func TestLego_RenewCert(t *testing.T) {
	if l == nil {
		t.Skip("V2BX_SKIP_NET_TESTS: 跳过联网证书测试")
	}
	log.Println(l.RenewCert())
}
