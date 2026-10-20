package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrSetJSONCoalescesConcurrentMisses(t *testing.T) {
	_, client := testRedisClient(t)
	ctx := context.Background()
	var loads atomic.Int32
	start := make(chan struct{})
	release := make(chan struct{})
	loader := func() (map[string]int, error) {
		loads.Add(1)
		close(start)
		<-release
		return map[string]int{"ok": 1}, nil
	}

	const callers = 8
	results := make([]map[string]int, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = GetOrSetJSON(ctx, client, "hot", time.Minute, loader)
		}(i)
	}
	<-start
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := loads.Load(); got != 1 {
		t.Fatalf("loader called %d times, want 1", got)
	}
	for i, err := range errs {
		if err != nil || results[i]["ok"] != 1 {
			t.Fatalf("result %d = %#v, err %v", i, results[i], err)
		}
	}
}

func TestGetOrSetJSONDoesNotCacheLoaderError(t *testing.T) {
	_, client := testRedisClient(t)
	ctx := context.Background()
	want := errors.New("temporary failure")
	var loads atomic.Int32
	_, err := GetOrSetJSON(ctx, client, "failing", time.Minute, func() (string, error) {
		loads.Add(1)
		return "", want
	})
	if !errors.Is(err, want) {
		t.Fatalf("first error = %v, want %v", err, want)
	}
	_, err = GetOrSetJSON(ctx, client, "failing", time.Minute, func() (string, error) {
		loads.Add(1)
		return "", want
	})
	if !errors.Is(err, want) {
		t.Fatalf("second error = %v, want %v", err, want)
	}
	if got := loads.Load(); got != 2 {
		t.Fatalf("loader called %d times, want 2 after failed flights", got)
	}
}

func TestGetOrSetJSONWaiterHonorsContextCancellation(t *testing.T) {
	_, client := testRedisClient(t)
	leaderDone := make(chan error, 1)
	start := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_, err := GetOrSetJSON(context.Background(), client, "cancel", time.Minute, func() (string, error) {
			close(start)
			<-release
			return "ok", nil
		})
		leaderDone <- err
	}()
	<-start
	waitCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetOrSetJSON(waitCtx, client, "cancel", time.Minute, func() (string, error) {
		t.Fatal("canceled waiter ran loader")
		return "", nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter error = %v, want context.Canceled", err)
	}
	close(release)
	if err := <-leaderDone; err != nil {
		t.Fatalf("leader error = %v", err)
	}
}
