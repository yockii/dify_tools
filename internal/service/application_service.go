package service

import (
	"context"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type applicationService struct {
	*BaseServiceImpl[*model.Application]
	appMap map[string]*model.Application
}

func NewApplicationService() *applicationService {
	srv := new(applicationService)
	srv.BaseServiceImpl = NewBaseService[*model.Application](BaseServiceConfig[*model.Application]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
		UpdateHook:     srv.UpdateHook,
		DeleteHook:     srv.DeleteHook,
	})
	srv.appMap = make(map[string]*model.Application)

	return srv
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
		logger.Error("查询记录失败", logger.F("error", err))
		return false, constant.ErrDatabaseError
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
		logger.Error("查询记录失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
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
		return constant.ErrRecordDuplicate
	}

	if err := s.db.Create(record).Error; err != nil {
		logger.Error("创建记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *applicationService) DeleteHook(ctx context.Context, record *model.Application) {
	delete(s.appMap, record.APIKey)
}

func (s *applicationService) UpdateHook(ctx context.Context, record *model.Application) {
	delete(s.appMap, record.APIKey)
}
