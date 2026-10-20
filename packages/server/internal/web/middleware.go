package web

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type originPolicy struct {
	exact map[string]bool
	wild  []wildcardOrigin
}

type wildcardOrigin struct {
	scheme string
	suffix string
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
	host, scheme := strings.ToLower(u.Hostname()), strings.ToLower(u.Scheme)
	for _, w := range p.wild {
		if w.scheme != "" && w.scheme != scheme {
			continue
		}
		if len(host) > len(w.suffix) && strings.HasSuffix(host, w.suffix) {
			return true
		}
	}
	return false
}

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
