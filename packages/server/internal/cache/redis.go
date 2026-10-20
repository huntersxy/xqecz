package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/redis/go-redis/v9"
)

const (
	SessionTTL      = 30 * 24 * time.Hour
	SessionRenew    = 15 * 24 * time.Hour
	viewKeyTTL      = 32 * 24 * time.Hour
	DefaultCacheTTL = 5 * time.Minute
)

var incrWithTTLScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n
`)

var getSessionScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if not value then
  return nil
end
local ttl = redis.call('PTTL', KEYS[1])
if ttl >= 0 and ttl < tonumber(ARGV[1]) then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return value
`)

type Client struct {
	rdb    *redis.Client
	prefix string

	flightMu    sync.Mutex
	jsonFlights map[string]*jsonFlight
}

type jsonFlight struct {
	done  chan struct{}
	value any
	err   error
}

func Open(cfg config.Config) *Client {
	return &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Host + ":" + strconv.Itoa(cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}),
		prefix: cfg.Redis.Prefix,
	}
}

func (c *Client) Raw() *redis.Client { return c.rdb }

func (c *Client) Key(key string) string { return c.prefix + key }

func (c *Client) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, c.Key(key)).Result()
}

func (c *Client) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.Key(key), val, ttl).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = c.Key(k)
	}
	return c.rdb.Del(ctx, full...).Err()
}

func (c *Client) SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.Key(key), b, ttl).Err()
}

func (c *Client) GetJSON(ctx context.Context, key string, out any) (bool, error) {
	raw, err := c.rdb.Get(ctx, c.Key(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, nil
	}
	return true, nil
}

func GetOrSetJSON[T any](ctx context.Context, c *Client, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var zero T
	if c == nil {
		return loader()
	}
	var cached T
	if hit, err := c.GetJSON(ctx, key, &cached); err == nil && hit {
		return cached, nil
	}

	flight, leader := c.beginJSONFlight(key)
	if !leader {
		select {
		case <-flight.done:
			if flight.err != nil {
				return zero, flight.err
			}
			value, ok := flight.value.(T)
			if !ok {
				return zero, fmt.Errorf("cache: in-flight value type mismatch for %q", key)
			}
			return value, nil
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	defer c.finishJSONFlight(key, flight)
	data, err := loader()
	if err != nil {
		flight.err = err
		return data, err
	}
	_ = c.SetJSON(ctx, key, data, ttl)
	flight.value = data
	return data, nil
}

func (c *Client) beginJSONFlight(key string) (*jsonFlight, bool) {
	c.flightMu.Lock()
	defer c.flightMu.Unlock()
	if c.jsonFlights == nil {
		c.jsonFlights = make(map[string]*jsonFlight)
	}
	if flight, ok := c.jsonFlights[key]; ok {
		return flight, false
	}
	flight := &jsonFlight{done: make(chan struct{})}
	c.jsonFlights[key] = flight
	return flight, true
}

func (c *Client) finishJSONFlight(key string, flight *jsonFlight) {
	c.flightMu.Lock()
	if current, ok := c.jsonFlights[key]; ok && current == flight {
		delete(c.jsonFlights, key)
	}
	close(flight.done)
	c.flightMu.Unlock()
}

func (c *Client) SetSession(ctx context.Context, sessionID string, userID uint64) error {
	return c.Set(ctx, "session:"+sessionID, userID, SessionTTL)
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (uint64, bool) {
	key := c.Key("session:" + sessionID)
	result, err := getSessionScript.Run(ctx, c.rdb, []string{key},
		SessionRenew.Milliseconds(), SessionTTL.Milliseconds()).Result()
	if err != nil || result == nil {
		return 0, false
	}
	var val string
	switch value := result.(type) {
	case string:
		val = value
	case []byte:
		val = string(value)
	default:
		return 0, false
	}
	uid, err := strconv.ParseUint(val, 10, 64)
	if err != nil || uid == 0 {
		return 0, false
	}
	return uid, true
}

func (c *Client) DelSession(ctx context.Context, sessionID string) error {
	return c.Del(ctx, "session:"+sessionID)
}

func (c *Client) IncrementView(ctx context.Context, contentID uint64) int64 {
	key := "views:date:" + time.Now().Format("2006-01-02") + ":" + strconv.FormatUint(contentID, 10)
	n, err := incrWithTTLScript.Run(ctx, c.rdb, []string{c.Key(key)}, viewKeyTTL.Milliseconds()).Int64()
	if err != nil {
		return 0
	}
	return n
}

type ScoredItem struct {
	ContentID uint64
	Score     float64
}

func (c *Client) WriteRecommendList(ctx context.Context, items []ScoredItem) error {
	if len(items) == 0 {
		return nil
	}
	temp := c.Key("recommend:hot:temp")
	final := c.Key("recommend:hot")
	members := make([]redis.Z, 0, len(items))
	for _, it := range items {
		members = append(members, redis.Z{Score: it.Score, Member: strconv.FormatUint(it.ContentID, 10)})
	}

	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, temp)
	pipe.ZAdd(ctx, temp, members...)
	pipe.Rename(ctx, temp, final)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) GetRecommendList(ctx context.Context, page, pageSize int) []uint64 {
	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize) - 1
	vals, err := c.rdb.ZRevRange(ctx, c.Key("recommend:hot"), start, end).Result()
	if err != nil {
		return nil
	}
	out := make([]uint64, 0, len(vals))
	for _, v := range vals {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (c *Client) GetRecommendTotal(ctx context.Context) int64 {
	n, err := c.rdb.ZCard(ctx, c.Key("recommend:hot")).Result()
	if err != nil {
		return 0
	}
	return n
}

func (c *Client) DelByPattern(ctx context.Context, pattern string) {
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, c.prefix+pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			c.rdb.Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func (c *Client) ClearCommentCache(ctx context.Context, contentID uint64) {
	id := strconv.FormatUint(contentID, 10)
	c.DelByPattern(ctx, "comments:"+id+":*")
	c.Del(ctx, "comment_count:"+id)
}

func (c *Client) ClearContentCache(ctx context.Context, contentID uint64) {
	c.Del(ctx, "content:"+strconv.FormatUint(contentID, 10))
}

func (c *Client) ClearContentListCache(ctx context.Context) {
	c.DelByPattern(ctx, "content_list:*")
	c.Del(ctx, "tags")
}

func (c *Client) ClearAllContentCaches(ctx context.Context) {
	c.DelByPattern(ctx, "content*")
	c.Del(ctx, "tags")
}

func (c *Client) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) int64 {
	n, err := incrWithTTLScript.Run(ctx, c.rdb, []string{c.Key(key)}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0
	}
	return n
}

func (c *Client) AcquireLock(ctx context.Context, key string, ttl time.Duration) bool {
	ok, err := c.rdb.SetNX(ctx, c.Key(key), "1", ttl).Result()
	return err == nil && ok
}

func (c *Client) ReleaseLock(ctx context.Context, key string) {
	c.rdb.Del(ctx, c.Key(key))
}

func (c *Client) RenewLock(ctx context.Context, key string, ttl time.Duration) bool {
	ok, err := c.rdb.Expire(ctx, c.Key(key), ttl).Result()
	return err == nil && ok
}
