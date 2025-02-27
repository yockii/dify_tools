package service

import (
	"github.com/yockii/dify_tools/internal/model"
	"gorm.io/gorm"
)

type columnInfoService struct {
	*BaseServiceImpl[*model.ColumnInfo]
}

func NewColumnInfoService() *columnInfoService {
	srv := new(columnInfoService)
	srv.BaseServiceImpl = NewBaseService[*model.ColumnInfo](BaseServiceConfig[*model.ColumnInfo]{
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
		return false, err
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
