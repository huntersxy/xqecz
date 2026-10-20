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

func Open(cfg config.Config) (*gorm.DB, error) {
	connectTimeout := cfg.MySQL.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = cfg.MySQL.ConnMaxLifetime
	}
	dsn := mysqldsn.Format(mysqldsn.Options{
		Host:      cfg.MySQL.Host,
		Port:      cfg.MySQL.Port,
		User:      cfg.MySQL.User,
		Password:  cfg.MySQL.Password,
		Database:  cfg.MySQL.Database,
		TLS:       cfg.MySQL.TLS,
		ParseTime: true,
		Timeout:   int(connectTimeout / time.Second),
	})

	gormCfg := &gorm.Config{
		NamingStrategy:         schema.NamingStrategy{SingularTable: true},
		Logger:                 logger.Default.LogMode(logger.Warn),
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
