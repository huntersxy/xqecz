// Command rediskeys 按模式列出业务 Redis 键，供迁移期核对缓存写入。
//
// 用法：go run ./cmd/rediskeys "content*"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/joho/godotenv"
)

func main() {
	pattern := "*"
	if len(os.Args) > 1 {
		pattern = os.Args[1]
	}
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))

	cfg := config.Load()
	c := cache.Open(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	full := cfg.Redis.Prefix + pattern
	var cursor uint64
	n := 0
	for {
		keys, next, err := c.Raw().Scan(ctx, cursor, full, 200).Result()
		if err != nil {
			log.Fatal(err)
		}
		for _, k := range keys {
			n++
			if n <= 40 {
				fmt.Println(k)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	fmt.Printf("共 %d 个键（模式 %s）\n", n, full)
}
