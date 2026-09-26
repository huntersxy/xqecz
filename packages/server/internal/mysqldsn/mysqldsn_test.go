package mysqldsn

import (
	"strings"
	"testing"
)

// TestFormatWithoutTLSMatchesLegacyShape 钉死「不加密」这条既有路径：
// 未配置 MYSQL_TLS 时必须与旧 DSN 一样不带 tls 参数，否则自建 MySQL 会因未知参数或
// 强制加密而连不上。
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

// TestFormatWithTLS 对应 TiDB Cloud 等强制加密的托管实例：
// tls 必须真的写进 DSN，否则服务端直接报 1105 insecure transport。
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

// TestFormatTimeout 覆盖连接超时：旧 DSN 用 MYSQL_CONNECT_TIMEOUT 换算，
// 迁移到本包后不能把该参数弄丢（否则建连会一直挂着而不报错）。
func TestFormatTimeout(t *testing.T) {
	got := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d", Timeout: 10})
	if !strings.Contains(got, "timeout=10s") {
		t.Fatalf("应包含 timeout=10s，实际 %q", got)
	}
	// 0 值表示交给驱动默认，不应写出 timeout=0s
	zero := Format(Options{Host: "h", Port: 3306, User: "u", Database: "d"})
	if strings.Contains(zero, "timeout=") {
		t.Fatalf("未指定超时时不应写 timeout，实际 %q", zero)
	}
}

// TestParseTimeToggles 逐字节搬运数据（迁移校验）时必须关闭 parseTime，
// 否则驱动会做时区换算，时间值被改写后校验必然不一致。
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
