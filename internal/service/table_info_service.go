package service

import (
	"context"
	"errors"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type tableInfoService struct {
	*BaseServiceImpl[*model.TableInfo]
}

func NewTableInfoService() *tableInfoService {
	srv := new(tableInfoService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.TableInfo]{
		NewModel:       srv.NewModel,
		BuildCondition: srv.BuildCondition,
	})
	return srv
}

func (s *tableInfoService) NewModel() *model.TableInfo {
	return &model.TableInfo{}
}

func (s *tableInfoService) BuildCondition(query *gorm.DB, record *model.TableInfo) *gorm.DB {
	if record.ID != 0 {
		query = query.Where("id = ?", record.ID)
	}
	if record.ApplicationID != 0 {
		query = query.Where("application_id = ?", record.ApplicationID)
	}
	if record.DataSourceID != 0 {
		query = query.Where("data_source_id = ?", record.DataSourceID)
	}
	if record.Name != "" {
		query = query.Where("name like ?", "%"+record.Name+"%")
	}
	if record.Comment != "" {
		query = query.Where("comment like ?", "%"+record.Comment+"%")
	}
	return query
}

func (s *tableInfoService) Delete(ctx context.Context, id uint64) error {
	var record model.TableInfo
	if err := s.db.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constant.ErrRecordNotFound
		}
		logger.Error("查询记录失败", logger.F("err", err))
		return constant.ErrDatabaseError
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.ColumnInfo{
			TableID: id,
		}).Error; err != nil {
			logger.Error("删除列信息失败", logger.F("err", err))
			return constant.ErrDatabaseError
		}
		if err := tx.Where("id = ?", id).Delete(&model.TableInfo{}).Error; err != nil {
			logger.Error("删除表信息失败", logger.F("err", err))
			return constant.ErrDatabaseError
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *tableInfoService) ListSchemaForDify(ctx context.Context, condition *model.TableInfo) ([]*model.TableInfo, error) {
	var list []*model.TableInfo
	if err := s.db.Select("ID", "Name", "Comment").
		Where(condition).Find(&list).Error; err != nil {
		logger.Error("查询表信息失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}
	return list, nil
}
