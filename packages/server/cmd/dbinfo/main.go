// Command dbinfo 打印当前 MySQL 账号的权限与库表规模，供迁移期排查使用。
package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/mysqldsn"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/joho/godotenv"
)

func main() {
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	cfg := config.Load()

	db, err := sql.Open("mysql", mysqldsn.Format(mysqldsn.Options{
		Host:     cfg.MySQL.Host,
		Port:     cfg.MySQL.Port,
		User:     cfg.MySQL.User,
		Password: cfg.MySQL.Password,
		Database: cfg.MySQL.Database,
		TLS:      cfg.MySQL.TLS,
		// 排查工具需要可读的 time.Time，故开启解析
		ParseTime: true,
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Printf("账号: %s  当前库: %s\n\n", cfg.MySQL.User, cfg.MySQL.Database)

	grows, err := db.Query("SHOW GRANTS")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("== GRANTS ==")
	for grows.Next() {
		var g string
		if err := grows.Scan(&g); err != nil {
			log.Fatal(err)
		}
		fmt.Println(g)
	}
	grows.Close()

	fmt.Println("\n== TABLES ==")
	rows, err := db.Query("SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name", cfg.MySQL.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var n sql.NullInt64
		if err := rows.Scan(&name, &n); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-20s ~%d 行\n", name, n.Int64)
	}
}
