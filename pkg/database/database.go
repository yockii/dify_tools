package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/yockii/dify_tools/pkg/config"
)

var db *gorm.DB

// Init 初始化数据库连接
func Init() error {
	var err error
	dbType := config.GetString("database.type")

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "t_", // 设置表名前缀
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用自动创建外键
	}

	switch dbType {
	case "postgres":
		db, err = gorm.Open(postgres.Open(config.GetDSN()), gormConfig)
	case "mysql":
		db, err = gorm.Open(mysql.Open(config.GetDSN()), gormConfig)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err != nil {
		return fmt.Errorf("connect to database failed: %v", err)
	}

	// 获取底层SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(config.GetInt("database.max_idle_conns"))
	sqlDB.SetMaxOpenConns(config.GetInt("database.max_open_conns"))
	sqlDB.SetConnMaxLifetime(time.Duration(config.GetInt("database.conn_max_lifetime")) * time.Second)
	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
