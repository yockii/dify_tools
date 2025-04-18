package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/datasource"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type dataSourceService struct {
	*BaseServiceImpl[*model.DataSource]
}

func NewDataSourceService() *dataSourceService {
	srv := new(dataSourceService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.DataSource]{
		NewModel:        srv.NewModel,
		CheckDuplicate:  srv.CheckDuplicate,
		DeleteCheck:     srv.DeleteCheck,
		BuildCondition:  srv.BuildCondition,
		ListOmitColumns: srv.ListOmitColumns,
	})
	return srv
}

func (s *dataSourceService) NewModel() *model.DataSource {
	return &model.DataSource{}
}

func (s *dataSourceService) CheckDuplicate(record *model.DataSource) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.DataSource{
		ApplicationID: record.ApplicationID,
		Name:          record.Name,
	})
	if record.ID != 0 {
		query = query.Where("id <> ?", record.ID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return false, constant.ErrDatabaseError
	}
	return count > 0, nil
}

func (s *dataSourceService) DeleteCheck(record *model.DataSource) error {
	return nil
}

func (s *dataSourceService) ListOmitColumns() []string {
	return []string{"password"}
}

func (s *dataSourceService) BuildCondition(query *gorm.DB, condition *model.DataSource) *gorm.DB {
	query = query.Where("application_id = ?", condition.ApplicationID)
	if condition.Name != "" {
		query = query.Where("name LIKE ?", "%"+condition.Name+"%")
	}
	if condition.Host != "" {
		query = query.Where("host LIKE ?", "%"+condition.Host+"%")
	}
	return query
}

func (s *dataSourceService) Delete(ctx context.Context, id uint64) error {
	var record model.DataSource
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constant.ErrRecordNotFound
		}
		logger.Error("查询记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	// 检查是否可以删除
	if err := s.DeleteCheck(&record); err != nil {
		return err
	}

	// 删除记录
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&record).Error; err != nil {
			logger.Error("删除记录失败", logger.F("error", err))
			return constant.ErrDatabaseError
		}
		// 删除表信息
		if err := tx.Where(&model.TableInfo{
			DataSourceID: record.ID,
		}).Delete(&model.TableInfo{}).Error; err != nil {
			logger.Error("删除表信息失败", logger.F("error", err))
			return constant.ErrDatabaseError
		}
		// 删除字段信息
		if err := tx.Where(&model.ColumnInfo{
			DataSourceID: record.ID,
		}).Delete(&model.ColumnInfo{}).Error; err != nil {
			logger.Error("删除字段信息失败", logger.F("error", err))
			return constant.ErrDatabaseError
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *dataSourceService) Sync(ctx context.Context, id uint64) error {
	// 查询数据源
	var dataSource model.DataSource
	if err := s.db.First(&dataSource, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constant.ErrRecordNotFound
		}
		logger.Error("查询数据源失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}

	db, err := datasource.GetDB(&dataSource)
	if err != nil {
		return err
	}

	// 查询表信息
	type TempTableInfo struct {
		TableName    string
		TableComment string
	}
	var tables []TempTableInfo
	switch dataSource.Type {
	case "mysql":
		err = db.Raw("SELECT table_name AS table_name, table_comment AS table_comment FROM information_schema.tables WHERE table_schema = ?", dataSource.Database).Scan(&tables).Error
	case "postgres":
		err = db.Raw("SELECT table_name, obj_description(table_name::regclass) AS table_comment FROM information_schema.tables WHERE table_catalog = ? AND table_schema = ?", dataSource.Database, dataSource.Schema).Scan(&tables).Error
	default:
		err = fmt.Errorf("unsupported database type: %s", dataSource.Type)
	}
	if err != nil {
		logger.Error("查询表信息失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}

	var tableInfos []*model.TableInfo
	for _, table := range tables {
		tableInfos = append(tableInfos, &model.TableInfo{
			ApplicationID: dataSource.ApplicationID,
			DataSourceID:  dataSource.ID,
			Name:          table.TableName,
			Comment:       table.TableComment,
		})
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&tableInfos).Error
		if err != nil {
			logger.Error("创建表信息失败", logger.F("error", err))
			return constant.ErrDatabaseError
		}

		for _, table := range tableInfos {
			if err := tx.Where(&model.TableInfo{
				DataSourceID: dataSource.ID,
				Name:         table.Name,
			}).Assign(table).FirstOrCreate(table).Error; err != nil {
				logger.Error("同步表信息失败", logger.F("error", err))
				return constant.ErrDatabaseError
			}
			// // 查询表信息
			// var tableInfo model.TableInfo
			// if err := tx.First(&tableInfo, "data_source_id = ? AND name = ?", dataSource.ID, table.Name).Error; err != nil {
			// 	logger.Error("查询表信息失败", logger.F("error", err))
			// 	return fmt.Errorf("查询表信息失败: %v", err)
			// }
			// if tableInfo.ID == 0 {
			// 	tx.Create(table)
			// } else {
			// 	tx.Model(&tableInfo).Updates(table)
			// 	table.ID = tableInfo.ID
			// }
		}

		migrator := db.Migrator()
		var columnInfos []*model.ColumnInfo
		for _, table := range tableInfos {
			ct, err := migrator.ColumnTypes(table.Name)
			if err != nil {
				logger.Error("查询列信息失败", logger.F("error", err))
				return constant.ErrDatabaseError
			}
			for _, c := range ct {
				t, _ := c.ColumnType()
				size, _ := c.Length()
				precision, scale, _ := c.DecimalSize()
				nullable, _ := c.Nullable()
				defaultValue, _ := c.DefaultValue()
				comment, _ := c.Comment()
				columnInfos = append(columnInfos, &model.ColumnInfo{
					ApplicationID: dataSource.ApplicationID,
					DataSourceID:  dataSource.ID,
					TableID:       table.ID,
					Name:          c.Name(),
					Type:          t,
					Size:          size,
					Precision:     precision,
					Scale:         scale,
					Nullable:      nullable,
					DefaultValue:  defaultValue,
					Comment:       comment,
				})
			}
		}

		for _, column := range columnInfos {
			// 查询字段信息
			if err := tx.Where(&model.ColumnInfo{
				DataSourceID: dataSource.ID,
				TableID:      column.TableID,
				Name:         column.Name,
			}).Assign(column).FirstOrCreate(column).Error; err != nil {
				logger.Error("同步字段信息失败", logger.F("error", err))
				return constant.ErrDatabaseError
			}
		}
		return nil
	})

	if err != nil {
		return constant.ErrDatabaseError
	}

	return nil
}

func (s *dataSourceService) ListForDify(ctx context.Context, condition *model.DataSource) ([]*model.DataSource, error) {
	var list []*model.DataSource
	condition.Status = 1
	if err := s.db.Where(condition).Select("ID", "Name", "Type").Find(&list).Error; err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return list, nil
}
