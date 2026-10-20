package mysqldsn

import (
	"strings"
	"testing"
)

func TestFormatWithoutTLSMatchesLegacyShape(t *testing.T) {
	got := Format(Options{
		Host: "127.0.0.1", Port: 3306,
		User: "root", Password: "pw", Database: "xqecz",
		ParseTime: true,
	})
	if strings.Contains(got, "tls=") {
		t.Fatalf("未配置 TLS 时不应出现 tls 参数，实际 %q", got)
	}
	for _, want := range []string{
		"root:pw@tcp(127.0.0.1:3306)/xqecz",
		"charset=utf8mb4",
		"parseTime=true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("DSN 缺少 %q，实际 %q", want, got)
		}
	}
}

func TestFormatWithTLS(t *testing.T) {
	for _, mode := range []string{"true", "skip-verify"} {
		got := Format(Options{
			Host: "gateway.example.com", Port: 4000,
			User: "u", Password: "p", Database: "xqecz",
			TLS: mode, ParseTime: true,
		})
		if !strings.Contains(got, "tls="+mode) {
			t.Fatalf("TLS=%s 应出现在 DSN 中，实际 %q", mode, got)
		}
	}
}

func TestFormatTimeout(t *testing.T) {
	got := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d", Timeout: 10})
	if !strings.Contains(got, "timeout=10s") {
		t.Fatalf("应包含 timeout=10s，实际 %q", got)
	}
	zero := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d"})
	if strings.Contains(zero, "timeout=") {
		t.Fatalf("未指定超时时不应写 timeout，实际 %q", zero)
	}
}

func TestParseTimeToggles(t *testing.T) {
	on := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d", ParseTime: true})
	if !strings.Contains(on, "parseTime=true") {
		t.Fatalf("parseTime 应为 true，实际 %q", on)
	}
	off := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d"})
	if strings.Contains(off, "parseTime=true") {
		t.Fatalf("parseTime 应为 false，实际 %q", off)
	}
}
