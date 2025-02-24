package datasource

import (
	"fmt"

	"github.com/yockii/dify_tools/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var mgr = &manager{
	dbMap: make(map[uint64]*gorm.DB),
}

type manager struct {
	dbMap map[uint64]*gorm.DB
}

func GetDB(ds *model.DataSource) (*gorm.DB, error) {
	if db, has := mgr.dbMap[ds.ID]; has {
		return db, nil
	}
	// 根据ds信息创建新的db
	db, err := createNewConnection(ds)
	if err != nil {
		return nil, err
	}
	mgr.dbMap[ds.ID] = db
	return db, nil
}

func createNewConnection(ds *model.DataSource) (*gorm.DB, error) {
	// 创建新的数据库连接
	switch ds.Type {
	case "mysql":
		// 创建 MySQL 连接
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			ds.User, ds.Password, ds.Host, ds.Port, ds.Database)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		return db, nil
	case "postgres":
		// 创建 Postgres 连接
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			ds.Host, ds.Port, ds.User, ds.Password, ds.Database)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		return db, nil
	}
	return nil, fmt.Errorf("unsupported database type: %s", ds.Type)
}
