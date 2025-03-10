package service

import (
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type agentService struct {
	*BaseServiceImpl[*model.Agent]
	appMap map[string]*model.Agent
}

func NewAgentService() *agentService {
	srv := new(agentService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.Agent]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
	})
	srv.appMap = make(map[string]*model.Agent)

	return srv
}

func (s *agentService) NewModel() *model.Agent {
	return &model.Agent{}
}

func (s *agentService) CheckDuplicate(record *model.Agent) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.Agent{
		Code: record.Code,
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

func (s *agentService) DeleteCheck(record *model.Agent) error {
	return nil
}

func (s *agentService) BuildCondition(query *gorm.DB, condition *model.Agent) *gorm.DB {
	if condition.Name != "" {
		query = query.Where("name LIKE ?", "%"+condition.Name+"%")
	}
	if condition.Code != "" {
		query = query.Where("code LIKE ?", "%"+condition.Code+"%")
	}
	return query
}
