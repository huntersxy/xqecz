package cache

import (
	"context"
	"encoding/json"
	"strconv"
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

// Client 包装 go-redis：统一给业务 key 加前缀（对齐旧 ioredis 的 keyPrefix 语义），
// 并在 Redis 不可用时对读路径降级（缓存故障不影响业务）。
type Client struct {
	rdb    *redis.Client
	prefix string
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

// Key 拼接带前缀的完整 key，业务代码只写逻辑 key。
func (c *Client) Key(key string) string { return c.prefix + key }

func (c *Client) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

// ---- 通用缓存 ----

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

// GetOrSetJSON 读穿缓存：命中直接返回，未命中执行 loader 并写回。
// Redis 不可用时退化为直查，与旧实现一致。
func GetOrSetJSON[T any](ctx context.Context, c *Client, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var cached T
	if hit, err := c.GetJSON(ctx, key, &cached); err == nil && hit {
		return cached, nil
	}
	data, err := loader()
	if err != nil {
		return data, err
	}
	_ = c.SetJSON(ctx, key, data, ttl)
	return data, nil
}

// ---- session ----

func (c *Client) SetSession(ctx context.Context, sessionID string, userID uint64) error {
	return c.Set(ctx, "session:"+sessionID, userID, SessionTTL)
}

// GetSession 读取会话，并在剩余 TTL 不足一半时滑动续期。
func (c *Client) GetSession(ctx context.Context, sessionID string) (uint64, bool) {
	key := c.Key("session:" + sessionID)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return 0, false
	}
	uid, err := strconv.ParseUint(val, 10, 64)
	if err != nil || uid == 0 {
		return 0, false
	}
	if ttl, err := c.rdb.TTL(ctx, key).Result(); err == nil && ttl >= 0 && ttl < SessionRenew {
		c.rdb.Expire(ctx, key, SessionTTL)
	}
	return uid, true
}

func (c *Client) DelSession(ctx context.Context, sessionID string) error {
	return c.Del(ctx, "session:"+sessionID)
}

// ---- 浏览量 ----

// IncrementView 记录当天浏览量，首次创建时设置 32 天过期。
func (c *Client) IncrementView(ctx context.Context, contentID uint64) int64 {
	key := "views:date:" + time.Now().Format("2006-01-02") + ":" + strconv.FormatUint(contentID, 10)
	n, err := c.rdb.Incr(ctx, c.Key(key)).Result()
	if err != nil {
		return 0
	}
	if n == 1 {
		c.rdb.Expire(ctx, c.Key(key), viewKeyTTL)
	}
	return n
}

// ---- 推荐 ZSet ----

type ScoredItem struct {
	ContentID uint64
	Score     float64
}

// WriteRecommendList 原子替换推荐位：先写临时 key 再 RENAME，读取端不会看到半写入状态。
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

// ---- 缓存失效 ----

// DelByPattern 用 SCAN 精确匹配带前缀的真实 key 后批量删除。
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

// ---- 限频与分布式锁 ----

// IncrWithTTL 自增计数并在首次创建时设置过期时间。
func (c *Client) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) int64 {
	n, err := c.rdb.Incr(ctx, c.Key(key)).Result()
	if err != nil {
		return 0
	}
	if n == 1 {
		c.rdb.Expire(ctx, c.Key(key), ttl)
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
