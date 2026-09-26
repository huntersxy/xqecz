package web

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// originPolicy 把 CORS_ORIGINS 编译成可判定的规则，支持两种写法：
//
//   - 精确 origin：`https://xq.xiey.work` —— 整串比对，与既有行为一致；
//   - 子域通配：`*.edgeone.cool` 或 `https://*.edgeone.cool` —— 按「主机名后缀」放行。
//     通配匹配任意深度子域（`a.edgeone.cool`、`a.b.edgeone.cool`），但不匹配裸域
//     （`edgeone.cool` 自身不算），且要求后缀前有点边界，避免 `notedgeone.cool`
//     这类借尾巴的域名混进来。写了 scheme 就必须一致，没写则不限 scheme。
//
// 通配是有意为之的放宽写法，只用于调试/预览域名（这类平台按项目动态分配子域，
// 事先无法枚举）：条目写在 .env 里一目了然，事后删掉该行即回到精确白名单。
type originPolicy struct {
	exact map[string]bool
	wild  []wildcardOrigin
}

// wildcardOrigin 是一条通配规则；scheme 为空表示不校验协议。
type wildcardOrigin struct {
	scheme string // 小写，如 "https"；空 = 任意
	suffix string // 小写，带前导点，如 ".edgeone.cool"
}

func newOriginPolicy(origins []string) originPolicy {
	p := originPolicy{exact: make(map[string]bool, len(origins))}
	for _, raw := range origins {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}
		rest, scheme := o, ""
		if i := strings.Index(o, "://"); i >= 0 {
			rest, scheme = o[i+3:], strings.ToLower(o[:i])
		}
		if s, ok := strings.CutPrefix(rest, "*."); ok && s != "" {
			p.wild = append(p.wild, wildcardOrigin{scheme: scheme, suffix: "." + strings.ToLower(s)})
			continue
		}
		p.exact[o] = true
	}
	return p
}

// allows 判定 Origin 是否在白名单内。空 origin 恒为 false。
func (p originPolicy) allows(origin string) bool {
	if origin == "" {
		return false
	}
	if p.exact[origin] {
		return true
	}
	if len(p.wild) == 0 {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	// Hostname() 已去掉端口；通配只认主机名，端口不参与比对。
	host, scheme := strings.ToLower(u.Hostname()), strings.ToLower(u.Scheme)
	for _, w := range p.wild {
		if w.scheme != "" && w.scheme != scheme {
			continue
		}
		// len 更长这条保证至少有一个标签在后缀之前，且后缀带点边界。
		if len(host) > len(w.suffix) && strings.HasSuffix(host, w.suffix) {
			return true
		}
	}
	return false
}

// CORS 与旧 NestJS 配置对齐：允许凭据，来源来自 CORS_ORIGINS 白名单。
func CORS(origins []string) gin.HandlerFunc {
	policy := newOriginPolicy(origins)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if policy.allows(origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type,X-API-Key,Authorization")
			h.Set("Vary", "Origin")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
