// Package cli 提供服务端二进制的运维子命令（不依赖 Go 工具链，部署机可直接调用）。
package cli

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunAdmin 创建管理员账号或重置其密码。
// 用法：xqecz-server admin -username admin [-email a@b.com] [-reset]
func RunAdmin(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("admin", flag.ExitOnError)
	username := fs.String("username", "admin", "管理员用户名")
	email := fs.String("email", "", "管理员邮箱（可空）")
	reset := fs.Bool("reset", false, "账号已存在时重置其密码")
	_ = fs.Parse(args)

	db, err := store.Open(cfg)
	if err != nil {
		log.Fatalf("连接数据库失败：%v", err)
	}

	password := randomPassword()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}

	var existing store.User
	err = db.Where("username = ?", *username).First(&existing).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		user := store.User{Username: *username, Password: string(hash), IsAdmin: 1}
		if *email != "" {
			user.Email = email
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("创建管理员失败：%v", err)
		}
		fmt.Printf("已创建管理员 #%d\n", user.ID)
	case err != nil:
		log.Fatalf("查询用户失败：%v", err)
	case !*reset:
		log.Fatalf("用户 %s 已存在；如需重置密码请加 -reset", *username)
	default:
		updates := map[string]any{"password": string(hash), "is_admin": 1, "is_banned": 0}
		if *email != "" {
			updates["email"] = *email
		}
		if err := db.Model(&store.User{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			log.Fatalf("重置密码失败：%v", err)
		}
		fmt.Printf("已重置管理员 #%d\n", existing.ID)
	}

	fmt.Println("========================================")
	fmt.Printf("用户名: %s\n", *username)
	fmt.Printf("密码:   %s\n", password)
	fmt.Println("密码仅显示这一次，请立即保存并登录修改。")
	fmt.Println("========================================")
}

func randomPassword() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
