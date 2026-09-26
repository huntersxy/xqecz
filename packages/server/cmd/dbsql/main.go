// Command dbsql 在项目库上执行一条 SQL，供迁移期排查与测试数据清理使用。
//
// 用法：go run ./cmd/dbsql "DELETE FROM users WHERE username = '__probe__'"
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/mysqldsn"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/joho/godotenv"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("用法：dbsql \"SQL\"")
	}
	stmt := strings.Join(args, " ")

	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	cfg := config.Load()

	db, err := sql.Open("mysql", mysqldsn.Format(mysqldsn.Options{
		Host:      cfg.MySQL.Host,
		Port:      cfg.MySQL.Port,
		User:      cfg.MySQL.User,
		Password:  cfg.MySQL.Password,
		Database:  cfg.MySQL.Database,
		TLS:       cfg.MySQL.TLS,
		ParseTime: true,
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	upper := strings.ToUpper(strings.TrimSpace(stmt))
	isQuery := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "DESCRIBE") ||
		strings.HasPrefix(upper, "DESC ")
	if isQuery {
		rows, err := db.Query(stmt)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Println(strings.Join(cols, "\t"))
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				log.Fatal(err)
			}
			parts := make([]string, len(vals))
			for i, v := range vals {
				switch t := v.(type) {
				case nil:
					parts[i] = "NULL"
				case []byte:
					parts[i] = string(t)
				default:
					parts[i] = fmt.Sprint(t)
				}
			}
			fmt.Println(strings.Join(parts, "\t"))
		}
		return
	}

	res, err := db.Exec(stmt)
	if err != nil {
		log.Fatal(err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("OK: %d 行受影响\n", n)
}
