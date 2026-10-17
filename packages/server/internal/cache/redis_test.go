package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func testRedisClient(t *testing.T) (*miniredis.Miniredis, *Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, &Client{rdb: rdb, prefix: "test:"}
}

func TestIncrementViewUsesDailyCounterAndExpiry(t *testing.T) {
	mr, c := testRedisClient(t)
	ctx := context.Background()

	if got := c.IncrementView(ctx, 42); got != 1 {
		t.Fatalf("first increment = %d, want 1", got)
	}
	if got := c.IncrementView(ctx, 42); got != 2 {
		t.Fatalf("second increment = %d, want 2", got)
	}
	key := "test:views:date:" + time.Now().Format("2006-01-02") + ":42"
	got, err := mr.Get(key)
	if err != nil || got != "2" {
		t.Fatalf("stored counter = %q, want 2", got)
	}
	if ttl := mr.TTL(key); ttl <= 0 || ttl > viewKeyTTL {
		t.Fatalf("counter TTL = %s, want a positive value <= %s", ttl, viewKeyTTL)
	}
}

func TestGetSessionRenewsNearExpiryInOneOperation(t *testing.T) {
	mr, c := testRedisClient(t)
	ctx := context.Background()
	if err := c.SetSession(ctx, "sid", 7); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(SessionTTL - SessionRenew + time.Minute)
	if uid, ok := c.GetSession(ctx, "sid"); !ok || uid != 7 {
		t.Fatalf("GetSession = (%d, %v), want (7, true)", uid, ok)
	}
	ttl := mr.TTL("test:session:sid")
	if ttl < SessionTTL-time.Minute || ttl > SessionTTL {
		t.Fatalf("renewed TTL = %s, want approximately %s", ttl, SessionTTL)
	}
	if _, ok := c.GetSession(ctx, "missing"); ok {
		t.Fatal("missing session unexpectedly authenticated")
	}
}

func TestIncrWithTTLOpensAndKeepsCounterExpiry(t *testing.T) {
	mr, c := testRedisClient(t)
	ctx := context.Background()
	if got := c.IncrWithTTL(ctx, "rate:user", 10*time.Minute); got != 1 {
		t.Fatalf("first increment = %d, want 1", got)
	}
	firstTTL := mr.TTL("test:rate:user")
	if got := c.IncrWithTTL(ctx, "rate:user", 10*time.Minute); got != 2 {
		t.Fatalf("second increment = %d, want 2", got)
	}
	secondTTL := mr.TTL("test:rate:user")
	if secondTTL <= 0 || secondTTL > firstTTL {
		t.Fatalf("counter TTL changed unexpectedly: first=%s second=%s", firstTTL, secondTTL)
	}
}
