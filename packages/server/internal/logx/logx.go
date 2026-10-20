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

var byteKeys = map[string]bool{
	"before": true, "after": true, "saved": true, "min_size": true,
	"orig": true, "got": true, "size": true, "file_size": true, "written": true,
}

func New(w io.Writer) slog.Handler {
	return newHandler(w, slog.LevelInfo, isTerminal(w), nil)
}

func NewWithBuffer(w io.Writer, buffer *Buffer) slog.Handler {
	return newHandler(w, slog.LevelInfo, isTerminal(w), buffer)
}

const defaultCapacity = 500

func newHandler(w io.Writer, minLevel slog.Level, color bool, buffer *Buffer) *Handler {
	return &Handler{out: w, minLevel: minLevel, color: color, buffer: buffer}
}

type Handler struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel slog.Level
	color    bool
	buffer   *Buffer
	pre      string
	group    string
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
	if h.buffer != nil {
		h.buffer.Append(Entry{Time: r.Time, Level: r.Level.String(), Message: r.Message, Line: string(stripANSI(b[:len(b)-1]))})
	}
	_, err := h.out.Write(b)
	return err
}

func stripANSI(src []byte) []byte {
	result := make([]byte, 0, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != 0x1b || i+1 >= len(src) || src[i+1] != '[' {
			result = append(result, src[i])
			continue
		}
		i += 2
		for i < len(src) {
			b := src[i]
			if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
				break
			}
			i++
		}
	}
	return result
}

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
	if isSensitiveKey(key) {
		return append(b, "[REDACTED]"...)
	}
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

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"password", "token", "secret", "api_key", "apikey", "authorization", "cookie"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

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
	return &Handler{out: h.out, minLevel: h.minLevel, color: h.color, buffer: h.buffer, pre: sb.String(), group: h.group}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	group := name
	if h.group != "" {
		group = h.group + "." + name
	}
	return &Handler{out: h.out, minLevel: h.minLevel, color: h.color, buffer: h.buffer, pre: h.pre, group: group}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
