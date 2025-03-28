package service

import (
	"context"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type columnInfoService struct {
	*BaseServiceImpl[*model.ColumnInfo]
}

func NewColumnInfoService() *columnInfoService {
	srv := new(columnInfoService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.ColumnInfo]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
	})
	return srv
}

func (s *columnInfoService) NewModel() *model.ColumnInfo {
	return &model.ColumnInfo{}
}

func (s *columnInfoService) CheckDuplicate(record *model.ColumnInfo) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.ColumnInfo{
		TableID: record.TableID,
		Name:    record.Name,
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

func (s *columnInfoService) DeleteCheck(record *model.ColumnInfo) error {
	return nil
}

func (s *columnInfoService) BuildCondition(query *gorm.DB, condition *model.ColumnInfo) *gorm.DB {
	if condition.ApplicationID != 0 {
		query = query.Where("application_id = ?", condition.ApplicationID)
	}
	if condition.DataSourceID != 0 {
		query = query.Where("data_source_id = ?", condition.DataSourceID)
	}
	if condition.TableID != 0 {
		query = query.Where("table_id = ?", condition.TableID)
	}
	if condition.Name != "" {
		query = query.Where("name LIKE ?", "%"+condition.Name+"%")
	}
	if condition.Comment != "" {
		query = query.Where("comment LIKE ?", "%"+condition.Comment+"%")
	}
	return query
}

func (s *columnInfoService) ListSchemaForDify(ctx context.Context, condition *model.ColumnInfo) ([]*model.ColumnInfo, error) {
	var cl []*model.ColumnInfo
	if err := s.db.Select("Name", "Type", "Comment").
		Where(condition).Find(&cl).Error; err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return cl, nil
}
