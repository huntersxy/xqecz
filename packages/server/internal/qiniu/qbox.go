package qiniu

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
)

// qboxSign 计算七牛 **QBox 路径签名**，赋给 Authorization（值已含 "QBox " 前缀外的部分）。
//
// QBox 与 Qiniu 是七牛并存的两套签名，**不可混用**：
//   - Qiniu：签名串含 Method/Path/Host/Content-Type/X-Qiniu-*，用于 kodo 的管理与统计接口；
//   - QBox ：签名串只有 `路径[?查询] + "\n"`，不含头也不含体，用于 fusion（CDN）接口。
//
// 见 developer.qiniu.com/fusion/13353/fusion-api-overview「鉴权方式」一栏。
func qboxSign(req *http.Request, accessKey, secretKey string) string {
	signingStr := req.URL.EscapedPath()
	if q := req.URL.RawQuery; q != "" {
		signingStr += "?" + q
	}
	signingStr += "\n"

	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(signingStr))
	return accessKey + ":" + base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// AuthorizeQbox 就地为 req 加上 QBox 凭证头。
//
// 导出它是因为证书管理（fusion.qiniuapi.com/sslcert）也走 QBox，
// 而与 kodo 的 Qiniu 签名分属两套、用错直接 401——调用方不该再拼一次。
func (s Signer) AuthorizeQbox(req *http.Request) {
	req.Header.Set("Authorization", "QBox "+qboxSign(req, s.AccessKey, s.SecretKey))
}
