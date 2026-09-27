package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/qiniu"
)

// TestLiveDeployCert 走一遍完整的换证书流程：
// 取当前证书 → 原样重新上传 → 绑定到域名 → 回收旧证书条目。
//
// 证书内容不变，所以对线上没有可见影响；价值在于把两条鉴权链路
// （fusion 的 QBox 上传/删除 + api 的 Qiniu 绑定）真的打一遍——
// 用错任何一个都是 401，而这两套签名在文档里只差一行表格。
//
// 私钥只落临时文件，任何路径都不打印。
func TestLiveDeployCert(t *testing.T) {
	ak := os.Getenv("QINIU_ACCESS_KEY")
	sk := os.Getenv("QINIU_SECRET_KEY")
	domain := os.Getenv("QINIU_CDN_DOMAIN")
	if ak == "" || sk == "" || domain == "" {
		t.Skip("缺少 QINIU_ACCESS_KEY / QINIU_SECRET_KEY / QINIU_CDN_DOMAIN")
	}
	signer := qiniu.Signer{AccessKey: ak, SecretKey: sk}

	prev := boundCertID(signer, domain)
	if prev == "" {
		t.Skip("域名当前没有绑定证书，无可复用材料")
	}

	// 取回证书与私钥（不打印）
	req, err := http.NewRequest(http.MethodGet, "https://fusion.qiniuapi.com/sslcert/"+prev, nil)
	if err != nil {
		t.Fatal(err)
	}
	signer.AuthorizeQbox(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("读取证书失败 status=%d", resp.StatusCode)
	}
	var payload struct {
		Cert struct {
			CA  string `json:"ca"`
			Pri string `json:"pri"`
		} `json:"cert"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("解析证书记录失败: %v", err)
	}
	if payload.Cert.CA == "" || payload.Cert.Pri == "" {
		t.Fatal("证书记录缺 ca/pri")
	}

	dir := t.TempDir()
	caPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	for path, data := range map[string]string{caPath: payload.Cert.CA, keyPath: payload.Cert.Pri} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Config{Qiniu: config.QiniuConfig{
		AccessKey:  ak,
		SecretKey:  sk,
		Bucket:     os.Getenv("QINIU_BUCKET"),
		Region:     os.Getenv("QINIU_REGION"),
		PublicBase: os.Getenv("QINIU_PUBLIC_BASE"),
	}}
	deployCert(cfg, caPath, keyPath)

	// 换绑后应指向新证书
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := boundCertID(signer, domain)
		if got != "" && got != prev {
			t.Logf("域名已换绑到新证书 certId=%s（旧 %s 的回收留给下一轮 prune：换绑下发期间它仍被标记为已绑定）", got, prev)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("控制面未反映换绑：prev=%s got=%s", prev, got)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
