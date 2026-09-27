package qiniu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLiveCertDetail 读证书状态，判断能否自动续签、还有多久到期。
//
// 只解出需要的字段：证书记录里带**私钥**（`pri`）与完整证书链（`ca`），
// 按原始响应打印必然把凭据甩进日志，所以这里用结构体白名单，从形状上杜绝外泄。
// 一次性排查工具。
func TestLiveCertDetail(t *testing.T) {
	ak := os.Getenv("QINIU_AK")
	sk := os.Getenv("QINIU_SK")
	certID := os.Getenv("QINIU_CERT_ID")
	if ak == "" || sk == "" || certID == "" {
		t.Skip("缺少 QINIU_AK / QINIU_SK / QINIU_CERT_ID")
	}
	s := Signer{AccessKey: ak, SecretKey: sk}

	// 证书管理走 fusion.qiniuapi.com + QBox 路径签名，与 kodo 的 Qiniu 签名不同。
	req, err := http.NewRequest(http.MethodGet, "https://fusion.qiniuapi.com/sslcert/"+certID, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.AuthorizeQbox(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("读取证书失败 status=%d", resp.StatusCode)
	}

	// 白名单结构体：未列出的字段（pri / ca 等）直接被丢弃。
	var payload struct {
		Cert struct {
			CertID     string   `json:"certid"`
			CommonName string   `json:"common_name"`
			DNSNames   []string `json:"dnsnames"`
			NotBefore  int64    `json:"not_before"`
			NotAfter   int64    `json:"not_after"`
			Product    string   `json:"product_short_name"`
			CertType   string   `json:"cert_type"`
			Enable     bool     `json:"enable"`
			AutoRenew  bool     `json:"auto_renew"`
			Renewable  bool     `json:"renewable"`
			State      string   `json:"state"`
		} `json:"cert"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("解析证书记录失败: %v", err)
	}
	c := payload.Cert
	fmt.Printf("证书       = %s\n", c.CertID)
	fmt.Printf("域名       = %s  %v\n", c.CommonName, c.DNSNames)
	fmt.Printf("产品       = %s / %s  enable=%v\n", c.Product, c.CertType, c.Enable)
	fmt.Printf("有效期     = %s ~ %s（还剩 %.1f 天）\n",
		fmtTime(c.NotBefore), fmtTime(c.NotAfter),
		time.Until(time.Unix(c.NotAfter, 0)).Hours()/24)
	fmt.Printf("自动续签   = auto_renew:%v  renewable:%v  state:%q\n",
		c.AutoRenew, c.Renewable, c.State)
}

func fmtTime(sec int64) string {
	if sec == 0 {
		return "-"
	}
	return time.Unix(sec, 0).In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04 MST")
}
