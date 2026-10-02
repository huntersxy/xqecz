package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/qiniu"
	"github.com/huntersxy/xqecz/server/internal/r2"
)

// challengePrefix 是 ACME HTTP-01 挑战文件在桶内的固定前缀。
//
// **刻意不加 QINIU_PREFIX**：ACME 回源请求的是根路径
// `/.well-known/acme-challenge/<token>`，带上 uploads/ 前缀会落到
// `uploads/.well-known/...` 而永远取不到，表现为校验 404。
const challengePrefix = ".well-known/acme-challenge/"

const acmeUsage = `用法：
  xqecz-server acme plant <token> <内容文件>   把挑战文件写进七牛桶并回读校验
  xqecz-server acme clean <token>              删除挑战文件
  xqecz-server acme deploy <fullchain> <key>   上传新证书并绑定到 CDN 域名
  xqecz-server acme prune                      回收已换绑的旧证书
  xqecz-server acme check                      打印七牛侧配置是否具备续签条件`

// RunAcme 是证书自动续签的辅助子命令。
//
// 背景：`img.xiey.work` 指向七牛 CDN、回源到桶 xy996，ACME 的 HTTP-01 挑战
// 请求根本到不了部署机。因此挑战文件由本子命令推进桶里，让 CDN 自己回源出来。
// 调用方（certbot 的 --manual-auth-hook / --manual-cleanup-hook）只负责传参。
func RunAcme(cfg config.Config, args []string) {
	if len(args) == 0 {
		log.Fatalf("%s", acmeUsage)
	}
	switch args[0] {
	case "plant":
		if len(args) != 3 {
			log.Fatalf("%s", acmeUsage)
		}
		plantChallenge(cfg, args[1], args[2])
	case "clean":
		if len(args) != 2 {
			log.Fatalf("%s", acmeUsage)
		}
		if err := s3Client(cfg).Delete(challengeKey(args[1])); err != nil {
			log.Fatalf("删除挑战文件失败：%v", err)
		}
	case "deploy":
		if len(args) != 3 {
			log.Fatalf("%s", acmeUsage)
		}
		deployCert(cfg, args[1], args[2])
	case "prune":
		pruneCerts(cfg)
	case "check":
		checkReady(cfg)
	default:
		log.Fatalf("%s", acmeUsage)
	}
}

// challengeKey 校验 token 并拼出桶内路径。
// ACME token 是 base64url 字符串，这里白名单校验——它直接进 URL 路径，
// 放过 `/`、`..` 等字符就等于给了一条任意对象读写的路。
func challengeKey(token string) string {
	if token == "" || len(token) > 256 {
		log.Fatalf("非法的挑战 token：长度异常")
	}
	for _, r := range token {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			log.Fatalf("非法的挑战 token：含不允许的字符 %q", r)
		}
	}
	return challengePrefix + token
}

// s3Client 用七牛凭据构造 S3 客户端；凭据不全直接退出（续签不可能成功，早失败早暴露）。
func s3Client(cfg config.Config) *r2.Client {
	q := cfg.Qiniu
	if !q.Enabled() {
		log.Fatalf("七牛凭据未配置（QINIU_*），无法完成证书续签")
	}
	client, err := r2.New(r2.Config{
		Endpoint:  q.Endpoint,
		AccessKey: q.AccessKey,
		SecretKey: q.SecretKey,
		Bucket:    q.Bucket,
		Region:    q.Region,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		log.Fatalf("构造七牛客户端失败：%v", err)
	}
	return client
}

// plantChallenge 把验证内容写进桶，并**回读 CDN 确认真的取得到**。
//
// 只写不读是不够的：ACME 校验失败时只回一句 403/404，看不出是签名错、
// 回源没配好还是缓存没生效。这里提前自己打一遍，把问题定位在这一步。
func plantChallenge(cfg config.Config, token, contentFile string) {
	key := challengeKey(token)
	if err := s3Client(cfg).Put(key, contentFile, "text/plain; charset=utf-8"); err != nil {
		log.Fatalf("写入挑战文件失败：%v", err)
	}

	base := strings.TrimRight(cfg.Qiniu.PublicBase, "/")
	if base == "" {
		log.Fatalf("QINIU_PUBLIC_BASE 未配置，无法校验挑战文件是否可访问")
	}
	url := base + "/" + key
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("回读挑战文件失败：%v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("回读挑战文件：status=%d url=%s", resp.StatusCode, url)
	}
	want, err := os.ReadFile(contentFile)
	if err != nil {
		log.Fatalf("读取本地挑战内容失败：%v", err)
	}
	if strings.TrimSpace(string(body)) != strings.TrimSpace(string(want)) {
		log.Fatalf("回读内容与写入不一致（可能命中了旧缓存）url=%s", url)
	}
	fmt.Printf("挑战文件就绪 %s\n", url)
}

// checkReady 打印续签所需的三项配置，便于部署后一次性确认。
func checkReady(cfg config.Config) {
	q := cfg.Qiniu
	fmt.Printf("七牛凭据     = %v\n", q.Enabled())
	fmt.Printf("桶 / 区域    = %s / %s\n", q.Bucket, q.Region)
	fmt.Printf("公开域名     = %s\n", q.PublicBase)
	fmt.Printf("CDN 加速域名 = %s\n", q.CDNDomain())
	if !q.Enabled() || q.PublicBase == "" {
		log.Fatalf("续签前置条件不满足")
	}
}

// deployCert 把新签发的证书上传七牛并绑定到 CDN 域名，两步缺一不可：
// 只上传不绑定，线上仍用旧证书；只绑定不上传，没有 certId 可绑。
//
// 两个接口分属两套鉴权——上传走 fusion + QBox，绑定走 api + Qiniu，
// 用错任意一个都是 401，这不是可选项。
//
// fullchain 是「服务器证书 + 中间证书」拼接的 PEM（certbot 的 fullchain.pem），
// key 是配套私钥（privkey.pem）。两者不配对会被 400324 直接拒绝。
func deployCert(cfg config.Config, fullchain, keyFile string) {
	q := cfg.Qiniu
	domain := q.CDNDomain()
	if !q.Enabled() || domain == "" {
		log.Fatalf("七牛凭据或 CDN 域名未配置，无法部署证书")
	}
	ca, err := os.ReadFile(fullchain)
	if err != nil {
		log.Fatalf("读取证书失败：%v", err)
	}
	pri, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("读取私钥失败：%v", err)
	}
	if len(bytes.TrimSpace(ca)) == 0 || len(bytes.TrimSpace(pri)) == 0 {
		log.Fatalf("证书或私钥内容为空")
	}

	signer := qiniu.Signer{AccessKey: q.AccessKey, SecretKey: q.SecretKey}

	// 先记下当前绑定的证书：换绑成功后它就没人用了，不删会一直攒，
	// 攒到上限会以 400500「证书数量超限」挡住下一次续签。
	prev := boundCertID(signer, domain)

	// 1) 上传：POST fusion.qiniuapi.com/sslcert（QBox）
	name := fmt.Sprintf("%s-%d", domain, time.Now().Unix())
	uploadBody, err := json.Marshal(map[string]string{
		"name": name,
		"pri":  string(pri),
		"ca":   string(ca),
	})
	if err != nil {
		log.Fatalf("序列化证书失败：%v", err)
	}
	upResp, err := requestJSON(signer, http.MethodPost,
		"https://fusion.qiniuapi.com/sslcert", uploadBody, true)
	if err != nil {
		log.Fatalf("上传证书失败：%v", err)
	}
	certID := jsonField(upResp, "certID", "certid")
	if certID == "" {
		log.Fatalf("上传证书未返回 certID：%s", clip(upResp))
	}

	// 2) 绑定：PUT api.qiniu.com/domain/<Name>/httpsconf（Qiniu）
	//    只传 certId——该接口是增量修改，未传项保留原配置（forceHttps/http2 等），
	//    全量回传反而可能把别人改过的设置顶掉。
	bindBody, err := json.Marshal(map[string]string{"certId": certID})
	if err != nil {
		log.Fatalf("序列化绑定参数失败：%v", err)
	}
	bindResp, err := requestJSON(signer, http.MethodPut,
		"https://api.qiniu.com/domain/"+domain+"/httpsconf", bindBody, false)
	if err != nil {
		log.Fatalf("绑定证书失败：%v", err)
	}
	if code := jsonField(bindResp, "code"); code != "200" {
		log.Fatalf("绑定证书被拒（code=%s）：%s", code, clip(bindResp))
	}
	fmt.Printf("证书已部署 domain=%s certId=%s\n（配置下发需 5-10 分钟，期间边缘仍用旧证书）\n", domain, certID)

	// 3) 回收旧证书：换绑后它已无人引用。失败只提示不致命——
	//    证书可能还绑着别的域名（400611），或压根没读到，都不该让整次续签判定为失败。
	if prev != "" && prev != certID {
		if err := deleteCert(signer, prev); err != nil {
			fmt.Printf("旧证书未回收（可手工清理）certId=%s err=%v\n", prev, err)
		} else {
			fmt.Printf("旧证书已回收 certId=%s\n", prev)
		}
	}
}

// boundCertID 读域名当前绑定的证书；读不到返回空串（不影响后续流程）。
func boundCertID(s qiniu.Signer, domain string) string {
	out, err := requestJSON(s, http.MethodGet, "https://api.qiniu.com/domain/"+domain, nil, false)
	if err != nil {
		return ""
	}
	var payload struct {
		HTTPS struct {
			CertID string `json:"certId"`
		} `json:"https"`
	}
	if json.Unmarshal(out, &payload) != nil {
		return ""
	}
	return payload.HTTPS.CertID
}

// deleteCert 删除一张证书（fusion + QBox）。
func deleteCert(s qiniu.Signer, certID string) error {
	req, err := http.NewRequest(http.MethodDelete, "https://fusion.qiniuapi.com/sslcert/"+certID, nil)
	if err != nil {
		return err
	}
	s.AuthorizeQbox(req)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, clip(out))
	}
	return nil
}

// pruneCerts 回收「已换绑、且不是刚签发」的旧证书。
//
// 为什么单独一个子命令：换绑成功后旧证书**不会立刻变为可删**——
// 配置要 5-10 分钟下发，期间证书侧仍认为它被域名占用（400611）。
// 所以 deploy 当场删必然失败，改由下一轮 cron 先 prune 再 renew。
// 不回收的话证书会一直攒，攒到上限以 400500 挡住下一次续签。
func pruneCerts(cfg config.Config) {
	q := cfg.Qiniu
	domain := q.CDNDomain()
	if !q.Enabled() || domain == "" {
		log.Fatalf("七牛凭据或 CDN 域名未配置")
	}
	signer := qiniu.Signer{AccessKey: q.AccessKey, SecretKey: q.SecretKey}
	bound := boundCertID(signer, domain)

	out, err := requestJSON(signer, http.MethodGet,
		"https://fusion.qiniuapi.com/sslcert?limit=100", nil, true)
	if err != nil {
		log.Fatalf("读取证书列表失败：%v", err)
	}
	var list struct {
		Certs []struct {
			CertID     string `json:"certid"`
			CreateTime int64  `json:"create_time"`
		} `json:"certs"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		log.Fatalf("解析证书列表失败：%v", err)
	}

	now := time.Now()
	var removed, skipped int
	for _, c := range list.Certs {
		if c.CertID == "" || c.CertID == bound {
			continue
		}
		// 24 小时内的不碰：可能正是上一次 deploy 刚换上去、控制面还在下发的那张。
		if now.Sub(time.Unix(c.CreateTime, 0)) < 24*time.Hour {
			skipped++
			continue
		}
		if err := deleteCert(signer, c.CertID); err != nil {
			skipped++
			fmt.Printf("跳过 certId=%s：%v\n", c.CertID, err)
			continue
		}
		removed++
	}
	// 一并报出列表规模：否则「列表为空」与「有若干条但全被过滤」看起来都是
	// 「跳过 0 张」，排查时只能靠猜。
	fmt.Printf("prune 完成：回收 %d 张，跳过 %d 张（列表共 %d 张，当前绑定 %s）\n",
		removed, skipped, len(list.Certs), bound)
}

// requestJSON 发一次带凭证的 JSON 请求。qbox 为 true 用 QBox 路径签名（fusion），
// 否则用 Qiniu 内容签名（api.qiniu.com）——两套签名不可混用。
func requestJSON(s qiniu.Signer, method, url string, body []byte, qbox bool) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if qbox {
		s.AuthorizeQbox(req)
	} else {
		s.Authorize(req, body)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("status=%d body=%s", resp.StatusCode, clip(out))
	}
	return out, nil
}

// jsonField 在响应里找第一个命中的字段名（七牛不同接口的大小写不统一：
// 上传返回 certID，读取返回 certid，绑定返回 code）。
func jsonField(raw []byte, keys ...string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				return s
			}
			return strings.Trim(string(v), `"`)
		}
	}
	return ""
}

// clip 截断响应体，避免把整份证书/私钥甩进日志。
func clip(b []byte) string {
	const max = 300
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}
