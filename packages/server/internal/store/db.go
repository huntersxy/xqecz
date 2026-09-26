package store

import (
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/mysqldsn"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Open 建立 MySQL 连接。DSN 与旧 TypeORM 连接参数对齐（utf8mb4 / parseTime / 本地时区），
// 加密模式由 cfg.MySQL.TLS 决定（托管实例如 TiDB Cloud 强制加密）。
func Open(cfg config.Config) (*gorm.DB, error) {
	dsn := mysqldsn.Format(mysqldsn.Options{
		Host:      cfg.MySQL.Host,
		Port:      cfg.MySQL.Port,
		User:      cfg.MySQL.User,
		Password:  cfg.MySQL.Password,
		Database:  cfg.MySQL.Database,
		TLS:       cfg.MySQL.TLS,
		ParseTime: true,
	})

	gormCfg := &gorm.Config{
		// 表名由各模型显式声明，禁用复数化推断，避免与既有表结构错位。
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Warn),
		// 关闭默认事务包裹（单条写操作无必要），减少一次往返。
		SkipDefaultTransaction: true,
	}

	db, err := gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.PoolSize)
	sqlDB.SetMaxIdleConns(cfg.MySQL.PoolSize)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}
