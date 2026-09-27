package qiniu

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 七牛时间戳防盗链的签名算法，见
// developer.qiniu.com/fusion/3841/timestamp-hotlinking-prevention-fusion
//
//	S     = key + url_encode(path) + T
//	SIGN  = lower_hex(md5(S))
//	T     = lower_hex(expiry_unix_time)          ← 必须是 16 进制，直接用十进制
//	                                                 会被 CDN 当成一个极大的过期时间，
//	                                                 等于完全没开鉴权
//	最终 URL = 原 URL + (&|?) + "sign=" + SIGN + "&t=" + T   ← sign 在前，t 在后
//
// url_encode 是「斜线不参与编码」的 UTF-8 百分号编码（大写十六进制），
// 与 Go 的 url.PathEscape 不同——后者会把 / 一起编掉。
const timeACLParamOrder = "sign"

// SignURL 给 URL 追加七牛时间戳防盗链参数。
// rawURL 须已含 scheme/host/path（可含 query，原样保留并追加参数）；
// key 是控制台配置的防盗链密钥；expiry 是该 URL 的过期时刻。
//
// path 取**解码后**再重新按「斜线不编码」编码的版本，保证与签名时逐字节一致——
// 签名串里的 path 和最终 URL 里的 path 必须是同一个字符串，否则必然 403。
func SignURL(rawURL, key string, expiry time.Time) (string, error) {
	if rawURL == "" || key == "" {
		return "", errors.New("qiniu: 签名需要完整的 URL 与密钥")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	t := strconv.FormatInt(expiry.Unix(), 16) // 十进制时间会被当成超大过期时间
	sum := md5.Sum([]byte(key + encodePath(u.Path) + t))
	sign := hex.EncodeToString(sum[:])

	q := u.RawQuery
	if q == "" {
		q = "sign=" + sign + "&t=" + t
	} else {
		q += "&sign=" + sign + "&t=" + t
	}
	u.RawQuery = q
	return u.String(), nil
}

// encodePath 实现七牛的 url_encode：保留 unreserved 字符与斜线，其余百分号编码（大写）。
func encodePath(p string) string {
	var b strings.Builder
	b.Grow(len(p))
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c == '/':
			b.WriteByte(c)
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(upperHex(c >> 4))
			b.WriteByte(upperHex(c & 0x0f))
		}
	}
	return b.String()
}

func upperHex(v byte) byte {
	if v < 10 {
		return '0' + v
	}
	return 'A' + (v - 10)
}

// SignTTL 是签名 URL 的默认有效期。
// 必须显著大于内容缓存 TTL（300s）：Redis 缓存的是**已签名**的 Item，
// 缓存里的 URL 若先于缓存过期，用户会拿到一串 403。
const SignTTL = time.Hour
