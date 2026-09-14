// Command dbinfo 打印当前 MySQL 账号的权限与库表规模，供迁移期排查使用。
package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/joho/godotenv"
)

func main() {
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	cfg := config.Load()

	c := mysql.NewConfig()
	c.User = cfg.MySQL.User
	c.Passwd = cfg.MySQL.Password
	c.Net = "tcp"
	c.Addr = fmt.Sprintf("%s:%d", cfg.MySQL.Host, cfg.MySQL.Port)
	c.DBName = cfg.MySQL.Database
	c.ParseTime = true
	c.Loc = time.Local
	c.Params = map[string]string{"charset": "utf8mb4"}

	db, err := sql.Open("mysql", c.FormatDSN())
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
