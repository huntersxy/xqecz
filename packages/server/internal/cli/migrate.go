package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/migrate"
	"github.com/huntersxy/xqecz/server/internal/store"
)

// RunMigrate applies embedded migrations. Without -status it is idempotent and
// safe to call on every deploy.
func RunMigrate(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	statusOnly := fs.Bool("status", false, "只查看已应用与待应用版本")
	timeout := fs.Duration("timeout", 5*time.Minute, "迁移超时时间")
	_ = fs.Parse(args)

	db, err := store.Open(cfg)
	if err != nil {
		log.Fatalf("连接数据库失败：%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if *statusOnly {
		applied, pending, err := migrate.Status(ctx, db)
		if err != nil {
			log.Fatalf("读取迁移状态失败：%v", err)
		}
		fmt.Printf("已应用 %d 个：\n", len(applied))
		for _, version := range applied {
			fmt.Println("  " + version)
		}
		if len(pending) > 0 {
			fmt.Printf("待应用 %d 个：\n", len(pending))
			for _, version := range pending {
				fmt.Println("  " + version)
			}
		}
		return
	}

	if err := migrate.Run(ctx, db); err != nil {
		log.Fatalf("迁移失败：%v", err)
	}
	fmt.Printf("迁移完成，共 %d 个版本可用\n", len(migrate.Versions()))
}
