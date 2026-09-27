package qiniu

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestLiveUsage 是真实环境校验：跑通当月用量的全部查询项。
// 未提供 QINIU_AK / QINIU_SK 时跳过，不进 CI。
//
//	QINIU_AK=... QINIU_SK=... QINIU_BUCKET=... [QINIU_CDN_DOMAIN=...] \
//	  go test ./internal/qiniu/ -run TestLiveUsage -v
func TestLiveUsage(t *testing.T) {
	ak := os.Getenv("QINIU_AK")
	sk := os.Getenv("QINIU_SK")
	if ak == "" || sk == "" {
		t.Skip("未提供 QINIU_AK / QINIU_SK，跳过")
	}
	c, err := New(Config{
		Signer:    Signer{AccessKey: ak, SecretKey: sk},
		Bucket:    os.Getenv("QINIU_BUCKET"),
		CDNDomain: os.Getenv("QINIU_CDN_DOMAIN"),
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := c.Usage(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("查询当月用量失败: %v", err)
	}
	fmt.Printf("存储       = %d Byte (%.2f MB)\n", u.StorageBytes, float64(u.StorageBytes)/1048576)
	fmt.Printf("CDN 回源   = %d Byte (%.2f MB)\n", u.OriginFlowBytes, float64(u.OriginFlowBytes)/1048576)
	fmt.Printf("外网流出   = %d Byte (%.2f MB)\n", u.EgressBytes, float64(u.EgressBytes)/1048576)
	fmt.Printf("CDN 下载   = %d Byte (%.2f MB) known=%v\n", u.CDNFlowBytes, float64(u.CDNFlowBytes)/1048576, u.CDNKnown)
	fmt.Printf("GET 次数   = %d\n", u.GETCount)
	fmt.Printf("PUT 次数   = %d\n", u.PUTCount)
}

// TestMonthWindow 校验自然月按北京时间划分，与部署机时区无关。
func TestMonthWindow(t *testing.T) {
	utc := time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC) // 北京时间已是 9/17 11:00
	begin, end := MonthWindow(utc)
	if begin.Location() != cst {
		t.Fatalf("窗口起点应落在北京时间，实际 %v", begin.Location())
	}
	if begin.Month() != time.September || begin.Day() != 1 || begin.Hour() != 0 {
		t.Fatalf("月初边界错误: %v", begin)
	}
	if !end.After(utc) {
		t.Fatalf("终点应晚于当前时刻，实际 %v <= %v", end, utc)
	}

	// 北京时间 8/31 23:00 == UTC 8/31 15:00，仍属 8 月
	lateAug := time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC)
	begin, _ = MonthWindow(lateAug)
	if begin.Month() != time.August {
		t.Fatalf("北京时间 8/31 应算 8 月，实际落在 %v", begin)
	}

	// 北京时间 9/1 00:30 == UTC 8/31 16:30，已属 9 月
	earlySep := time.Date(2026, 8, 31, 16, 30, 0, 0, time.UTC)
	begin, _ = MonthWindow(earlySep)
	if begin.Month() != time.September {
		t.Fatalf("北京时间 9/1 应算 9 月，实际落在 %v", begin)
	}
}
