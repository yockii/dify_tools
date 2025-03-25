package service

import (
	"context"
	"errors"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type applicationService struct {
	*BaseServiceImpl[*model.Application]
	dictService DictService
	appMap      map[string]*model.Application
	appAgentMap map[uint64]map[uint64]*model.ApplicationAgent
}

func NewApplicationService(
	dictService DictService,
) *applicationService {
	srv := new(applicationService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.Application]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
		UpdateHook:     srv.UpdateHook,
		DeleteHook:     srv.DeleteHook,
	})
	srv.dictService = dictService
	srv.appMap = make(map[string]*model.Application)
	srv.appAgentMap = make(map[uint64]map[uint64]*model.ApplicationAgent)

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
	err := s.db.Where(map[string]interface{}{
		"api_key": apiKey,
	}).First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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

func (s *applicationService) ApplicationAgents(ctx context.Context, applicationID uint64) ([]*model.ApplicationAgent, error) {
	var agents []*model.ApplicationAgent
	err := s.db.Where("application_id = ?", applicationID).
		Preload("Agent").Find(&agents).Error
	if err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return agents, nil
}

func (s *applicationService) AddApplicationAgent(ctx context.Context, applicationID, agentID uint64) error {
	// 先查重
	var count int64
	if err := s.db.Model(&model.ApplicationAgent{}).Where("application_id = ? AND agent_id = ?", applicationID, agentID).Count(&count).Error; err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	if count > 0 {
		return nil
	}

	if err := s.db.Create(&model.ApplicationAgent{
		ApplicationID: applicationID,
		AgentID:       agentID,
	}).Error; err != nil {
		logger.Error("创建记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *applicationService) DeleteApplicationAgent(ctx context.Context, applicationID, agentID uint64) error {
	if err := s.db.Where("application_id = ? AND agent_id = ?", applicationID, agentID).
		Delete(&model.ApplicationAgent{}).Error; err != nil {
		logger.Error("删除记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *applicationService) GetApplicationAgent(ctx context.Context, applicationID, agentID uint64) (*model.ApplicationAgent, error) {
	// 先从缓存中获取
	if agentMap, ok := s.appAgentMap[applicationID]; ok {
		if agent, ok := agentMap[agentID]; ok {
			return agent, nil
		}
	} else {
		s.appAgentMap[applicationID] = make(map[uint64]*model.ApplicationAgent)
	}

	// 已经确保s.appAgentMap[applicationID]存在
	agentMap := s.appAgentMap[applicationID]
	useDefaultAgent := agentID == 0

	var agent model.ApplicationAgent
	if useDefaultAgent {
		// 未指定智能体，使用默认智能体
		defaultAgentIDDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyDefaultAgentID)
		if err != nil {
			logger.Error("获取字典值失败", logger.F("err", err))
			return nil, err
		}
		if defaultAgentIDDict == nil || defaultAgentIDDict.Value == "" {
			logger.Warn("未配置默认智能体", logger.F("dict_id", defaultAgentIDDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		agentID = defaultAgentIDDict.ValueUint64()
		if agentID == 0 {
			logger.Warn("默认智能体ID配置错误", logger.F("dict_id", defaultAgentIDDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		var defaultAgentInstance model.Agent
		err = s.db.Where("id = ?", agentID).Take(&defaultAgentInstance).Error
		if err != nil {
			logger.Error("查询记录失败", logger.F("error", err))
			return nil, constant.ErrDatabaseError
		}
		agent = model.ApplicationAgent{
			ApplicationID: applicationID,
			AgentID:       agentID,
			Agent:         &defaultAgentInstance,
		}
	} else {
		err := s.db.Where(&model.ApplicationAgent{
			AgentID: agentID,
		}).Where("application_id = ?", applicationID).Preload("Agent").
			First(&agent).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			logger.Error("查询记录失败", logger.F("error", err))
			return nil, constant.ErrDatabaseError
		}
	}
	agentMap[agentID] = &agent
	if useDefaultAgent {
		// 缓存默认智能体
		agentMap[0] = &agent
	}
	return &agent, nil
}
