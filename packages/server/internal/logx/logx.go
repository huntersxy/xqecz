// Package logx 提供比 slog TextHandler 更适合人眼阅读的控制台日志格式：
//
//	2026-09-14 22:09:49 INFO  tinypng 压缩完成  id=844 file=db36….webp before=1.36MB after=92.9KB saved=1.27MB
//	2026-09-14 22:22:29 WARN  http  req=GET / status=404 cost=0.1ms
//
// 设计取舍：单行、无引号噪音；字节类字段自动转人类可读；duration 转 ms/µs；
// 输出为终端时上色，重定向到文件（生产 `>> log`）时自动纯文本。
package logx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// byteKeys 的值按字节数格式化（如 1427370 → 1.36MB）。
var byteKeys = map[string]bool{
	"before": true, "after": true, "saved": true, "min_size": true,
	"orig": true, "got": true, "size": true, "file_size": true, "written": true,
}

// New 创建默认配置的 Handler（Info 起记，stdout 为终端时带色）。
func New(w io.Writer) slog.Handler {
	return newHandler(w, slog.LevelInfo, isTerminal(w))
}

func newHandler(w io.Writer, minLevel slog.Level, color bool) *Handler {
	return &Handler{out: w, minLevel: minLevel, color: color}
}

type Handler struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel slog.Level
	color    bool
	pre      string // WithAttrs/WithGroup 累积的已格式化前缀
	group    string // 当前组前缀（仅 WithGroup）
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.minLevel }

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b []byte
	b = append(b, '[')
	b = appendTime(b, r.Time, h.color)
	b = append(b, ' ')
	b = appendLevel(b, r.Level, h.color)
	b = append(b, ']', ' ')
	b = append(b, r.Message...)
	b = h.appendAttrs(b, r)
	b = append(b, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(b)
	return err
}

// appendAttrs 依次输出：WithAttrs 累积前缀 → 本条记录的属性。组内键带组名前缀。
func (h *Handler) appendAttrs(b []byte, r slog.Record) []byte {
	if h.pre != "" {
		b = append(b, h.pre...)
	}
	r.Attrs(func(a slog.Attr) bool {
		b = appendGrouped(b, h.group, a, h.color)
		return true
	})
	return b
}

func appendGrouped(b []byte, group string, a slog.Attr, color bool) []byte {
	if a.Equal(slog.Attr{}) {
		return b
	}
	if a.Value.Kind() == slog.KindGroup {
		grp := a.Value.Group()
		if len(grp) == 0 {
			return b
		}
		sub := a.Key
		if group != "" {
			sub = group + "." + sub
		}
		for _, ga := range grp {
			b = appendGrouped(b, sub, ga, color)
		}
		return b
	}
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	b = append(b, ' ', ' ')
	b = appendKey(b, key, color)
	b = append(b, '=')
	return appendValue(b, a.Value, key, color)
}

func appendTime(b []byte, t time.Time, color bool) []byte {
	s := t.Format("2006-01-02 15:04:05")
	if color {
		return append(b, "\x1b[90m"+s+"\x1b[0m"...)
	}
	return append(b, s...)
}

// appendLevel 输出 4 字符级别名（不带尾随空格，方括号由 Handle 统一包裹）。
func appendLevel(b []byte, l slog.Level, color bool) []byte {
	name, code := "INFO", "32"
	switch {
	case l >= slog.LevelError:
		name, code = "ERRO", "31"
	case l >= slog.LevelWarn:
		name, code = "WARN", "33"
	case l < slog.LevelInfo:
		name, code = "DEBU", "90"
	}
	if color {
		return append(b, "\x1b["+code+"m"+name+"\x1b[0m"...)
	}
	return append(b, name...)
}

func appendKey(b []byte, k string, color bool) []byte {
	if color {
		return append(b, "\x1b[36m"+k+"\x1b[0m"...)
	}
	return append(b, k...)
}

func appendValue(b []byte, v slog.Value, key string, color bool) []byte {
	var s string
	switch v.Kind() {
	case slog.KindDuration:
		s = formatDuration(v.Duration())
	case slog.KindInt64:
		n := v.Int64()
		if byteKeys[key] {
			s = formatBytes(n)
		} else {
			s = fmt.Sprint(n)
		}
	case slog.KindUint64:
		n := v.Uint64()
		if byteKeys[key] {
			s = formatBytes(int64(n))
		} else {
			s = fmt.Sprint(n)
		}
	case slog.KindFloat64:
		s = fmt.Sprint(v.Float64())
	case slog.KindTime:
		s = v.Time().Format("2006-01-02 15:04:05")
	case slog.KindGroup:
		s = fmt.Sprint(v.Group())
	default:
		s = v.String()
		if strings.ContainsAny(s, " \t") {
			s = fmt.Sprintf("%q", s)
		}
	}
	return append(b, s...)
}

// formatDuration：亚毫秒用 µs、毫秒 1 位小数、秒级 2 位小数、分钟以上用 m。
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Minute:
		return fmt.Sprintf("%.1fm", d.Minutes())
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.0fµs", float64(d)/float64(time.Microsecond))
	}
}

// formatBytes：1023 → 1023B；1.36MB 风格。
func formatBytes(n int64) string {
	switch {
	case n < 0:
		return "?" + formatBytes(-n)
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(n)/(1024))
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.2fMB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.2fGB", float64(n)/(1024*1024*1024))
	}
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var sb strings.Builder
	sb.WriteString(h.pre)
	for _, a := range attrs {
		if a.Equal(slog.Attr{}) {
			continue
		}
		if a.Value.Kind() == slog.KindGroup {
			for _, ga := range a.Value.Group() {
				gkey := a.Key
				if h.group != "" {
					gkey = h.group + "." + gkey
				}
				sb.Write(appendGrouped(nil, gkey, ga, h.color))
			}
			continue
		}
		sb.Write(appendGrouped(nil, h.group, a, h.color))
	}
	return &Handler{out: h.out, minLevel: h.minLevel, color: h.color, pre: sb.String(), group: h.group}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	group := name
	if h.group != "" {
		group = h.group + "." + name
	}
	return &Handler{out: h.out, minLevel: h.minLevel, color: h.color, pre: h.pre, group: group}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
