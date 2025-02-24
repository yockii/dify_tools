package service

import (
	"context"
	"fmt"

	"github.com/yockii/dify_tools/internal/model"
	"gorm.io/gorm"
)

type applicationService struct {
	*BaseService[*model.Application]
	appMap map[string]*model.Application
}

func NewApplicationService() *applicationService {
	return &applicationService{
		NewBaseService[*model.Application](),
		make(map[string]*model.Application),
	}
}

func (s *applicationService) NewModel() *model.Application {
	return &model.Application{}
}

func (s *applicationService) CheckDuplicate(record *model.Application) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.Application{
		Name: record.Name,
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

func (s *applicationService) DeleteCheck(record *model.Application) error {
	return nil
}

func (s *applicationService) BuildCondition(query *gorm.DB, condition *model.Application) *gorm.DB {
	if condition.Name != "" {
		query = query.Where("name LIKE ?", "%"+condition.Name+"%")
	}
	if condition.Status != 0 {
		query = query.Where("status = ?", condition.Status)
	}
	return query
}

func (s *applicationService) GetByApiKey(ctx context.Context, apiKey string) (*model.Application, error) {
	if app, ok := s.appMap[apiKey]; ok {
		return app, nil
	}
	var app model.Application
	err := s.db.Where(&model.Application{APIKey: apiKey}).First(&app).Error
	if err != nil {
		return nil, err
	}
	s.appMap[apiKey] = &app
	return &app, nil
}

func (s *applicationService) Create(ctx context.Context, record *model.Application) error {
	// 检查是否重复
	duplicate, err := s.CheckDuplicate(record)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("记录已存在")
	}

	if err := s.db.Create(record).Error; err != nil {
		return fmt.Errorf("创建记录失败: %v", err)
	}
	return nil
}

func (s *applicationService) DeleteHook(ctx context.Context, record *model.Application) {
	delete(s.appMap, record.APIKey)
}

func (s *applicationService) UpdateHook(ctx context.Context, record *model.Application) {
	delete(s.appMap, record.APIKey)
}
