package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/logx"
	"github.com/huntersxy/xqecz/server/internal/web"
)

const (
	defaultLogLimit = 200
	maxLogLimit     = 500
)

type logEntryResponse struct {
	ID      uint64   `json:"id"`
	Time    web.Time `json:"time"`
	Level   string   `json:"level"`
	Message string   `json:"message"`
	Line    string   `json:"line"`
}

// logs exposes only the bounded in-memory diagnostic stream. It deliberately
// has no file or database fallback: a missing buffer means this process was
// started without the optional viewer, and the endpoint returns an empty set.
func (h *Handler) logs(c *gin.Context) {
	limit := atoiDefault(c.Query("limit"), defaultLogLimit)
	if limit < 1 {
		limit = defaultLogLimit
	}
	if limit > maxLogLimit {
		limit = maxLogLimit
	}
	after, _ := strconv.ParseUint(strings.TrimSpace(c.Query("after")), 10, 64)
	level := strings.ToUpper(strings.TrimSpace(c.Query("level")))
	if level != "" && level != "DEBUG" && level != "INFO" && level != "WARN" && level != "ERROR" {
		web.Fail(c, 400, "日志级别不合法")
		return
	}

	var entries []logx.Entry
	var latest uint64
	var truncated bool
	if h.deps.Logs != nil {
		entries, latest, truncated = h.deps.Logs.Snapshot(after, maxLogLimit)
	}
	filtered := make([]logEntryResponse, 0, minInt(limit, len(entries)))
	for _, entry := range entries {
		if level != "" && entry.Level != level {
			continue
		}
		filtered = append(filtered, logEntryResponse{
			ID: entry.ID, Time: web.TimeOf(entry.Time), Level: strings.ToLower(entry.Level),
			Message: entry.Message, Line: entry.Line,
		})
		if len(filtered) == limit {
			break
		}
	}
	if filtered == nil {
		filtered = []logEntryResponse{}
	}
	web.OK(c, gin.H{
		"entries":     filtered,
		"next_cursor": latest,
		"truncated":   truncated,
		"server_time": web.TimeOf(time.Now()),
	}, "ok")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
