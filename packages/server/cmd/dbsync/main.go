// Command dbsync 把源库结构（可选数据）复制到目标库，用于本地开发影子库。
//
// 用法：
//
//	go run ./cmd/dbsync -target xqv2_dev            # 仅结构
//	go run ./cmd/dbsync -target xqv2_dev -with-data # 结构 + 数据
//
// 源库连接信息取自项目根 .env（MYSQL_*、MYSQL_DATABASE）。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/mysqldsn"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/joho/godotenv"
)

func main() {
	target := flag.String("target", "", "目标库名（必填）")
	withData := flag.Bool("with-data", false, "同时复制表数据")
	maxRows := flag.Int64("max-rows", 200000, "单表超过该行数时跳过数据复制")
	flag.Parse()

	if *target == "" {
		log.Fatal("-target 必填")
	}

	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	cfg := config.Load()
	src := cfg.MySQL.Database

	if src == *target {
		log.Fatalf("目标库不能与源库相同（%s）", src)
	}

	db, err := sql.Open("mysql", dsn(cfg, ""))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)

	q := func(s string) string { return "`" + s + "`" }

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + q(*target) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		log.Fatalf("创建目标库失败：%v", err)
	}

	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name", src)
	if err != nil {
		log.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			log.Fatal(err)
		}
		tables = append(tables, t)
	}
	rows.Close()
	if len(tables) == 0 {
		log.Fatalf("源库 %s 没有表", src)
	}

	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		log.Fatal(err)
	}

	copied := 0
	var skipped []string
	for _, t := range tables {
		var name, ddl string
		if err := db.QueryRow("SHOW CREATE TABLE "+q(src)+"."+q(t)).Scan(&name, &ddl); err != nil {
			log.Fatalf("读取 %s 结构失败：%v", t, err)
		}
		if _, err := db.Exec("DROP TABLE IF EXISTS " + q(*target) + "." + q(t)); err != nil {
			log.Fatal(err)
		}
		if _, err := db.Exec(ddl); err != nil {
			log.Fatalf("在目标库创建 %s 失败：%v", t, err)
		}

		if !*withData {
			continue
		}
		var n int64
		if err := db.QueryRow("SELECT COUNT(*) FROM " + q(src) + "." + q(t)).Scan(&n); err != nil {
			log.Fatal(err)
		}
		if n > *maxRows {
			skipped = append(skipped, fmt.Sprintf("%s(%d 行)", t, n))
			continue
		}
		if _, err := db.Exec("INSERT INTO " + q(*target) + "." + q(t) + " SELECT * FROM " + q(src) + "." + q(t)); err != nil {
			log.Fatalf("复制 %s 数据失败：%v", t, err)
		}
		copied++
	}

	fmt.Printf("目标库 %s：%d 张表结构已同步", *target, len(tables))
	if *withData {
		fmt.Printf("，%d 张表数据已复制", copied)
	}
	fmt.Println()
	if len(skipped) > 0 {
		fmt.Println("跳过大表：" + strings.Join(skipped, ", "))
	}
}

func dsn(cfg config.Config, database string) string {
	return mysqldsn.Format(mysqldsn.Options{
		Host:      cfg.MySQL.Host,
		Port:      cfg.MySQL.Port,
		User:      cfg.MySQL.User,
		Password:  cfg.MySQL.Password,
		Database:  database,
		TLS:       cfg.MySQL.TLS,
		ParseTime: true,
	})
}
