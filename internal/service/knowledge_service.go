package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/dify"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

type knowledgeBaseService struct {
	*BaseServiceImpl[*model.KnowledgeBase]
	applicationService ApplicationService
	dictService        DictService
}

func NewKnowledgeBaseService(dictService DictService, applicationService ApplicationService) *knowledgeBaseService {
	srv := new(knowledgeBaseService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.KnowledgeBase]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
	})
	srv.applicationService = applicationService
	srv.dictService = dictService
	return srv
}

func (s *knowledgeBaseService) NewModel() *model.KnowledgeBase {
	return &model.KnowledgeBase{}
}

func (s *knowledgeBaseService) CheckDuplicate(record *model.KnowledgeBase) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.KnowledgeBase{
		ApplicationID:     record.ApplicationID,
		KnowledgeBaseName: record.KnowledgeBaseName,
	})
	query = query.Where("custom_id = ?", record.CustomID)

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

func (s *knowledgeBaseService) DeleteCheck(record *model.KnowledgeBase) error {
	return nil
}

func (s *knowledgeBaseService) BuildCondition(query *gorm.DB, condition *model.KnowledgeBase) *gorm.DB {
	if condition.ApplicationID != 0 {
		query = query.Where("application_id = ?", condition.ApplicationID)
	}
	if condition.KnowledgeBaseName != "" {
		query = query.Where("knowledge_name LIKE ?", "%"+condition.KnowledgeBaseName+"%")
	}
	return query
}

func (s *knowledgeBaseService) GetByOuterID(outerID string) (*model.KnowledgeBase, error) {
	var knowledge model.KnowledgeBase
	err := s.db.Where(&model.KnowledgeBase{OuterID: outerID}).First(&knowledge).Error
	if err != nil {
		logger.Error("获取知识库失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}
	return &knowledge, nil
}

func (s *knowledgeBaseService) GetByApplicationID(applicationID uint64) ([]*model.KnowledgeBase, error) {
	var list []*model.KnowledgeBase
	err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID}).Find(&list).Error
	if err != nil {
		logger.Error("获取知识库列表失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}
	return list, nil
}

func (s *knowledgeBaseService) GetByApplicationIDAndKnowledgeName(applicationID uint64, knowledgeName string) (*model.KnowledgeBase, error) {
	var knowledge model.KnowledgeBase
	err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID, KnowledgeBaseName: knowledgeName}).First(&knowledge).Error
	if err != nil {
		logger.Error("获取知识库失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}
	return &knowledge, nil
}

func (s *knowledgeBaseService) GetDifyKnowledgeBaseClient(ctx context.Context) (*dify.KnowledgeBaseClient, error) {
	kbClient := dify.GetDefaultKnowledgeBaseClient()
	if kbClient == nil {
		difyBaseUrlDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyBaseUrl)
		if err != nil {
			logger.Error("获取dify接口地址失败", logger.F("err", err))
			return nil, err
		}
		if difyBaseUrlDict == nil || difyBaseUrlDict.Value == "" {
			logger.Warn("未配置dify接口地址", logger.F("dict_id", difyBaseUrlDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		difyBaseUrl := difyBaseUrlDict.Value
		difyDatasetsTokenDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyToken)
		if err != nil {
			return nil, err
		}
		if difyDatasetsTokenDict == nil || difyDatasetsTokenDict.Value == "" {
			logger.Warn("未配置dify知识库API密钥", logger.F("dict_id", difyDatasetsTokenDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		difyDatasetsToken := difyDatasetsTokenDict.Value
		kbClient = dify.InitDefaultKnowledgeBaseClient(difyBaseUrl, difyDatasetsToken)
	}
	return kbClient, nil
}

func (s *knowledgeBaseService) Create(ctx context.Context, knowledgeBase *model.KnowledgeBase) error {
	if knowledgeBase.ApplicationID == 0 && knowledgeBase.CustomID == "" {
		return constant.ErrInvalidParams
	}
	appName := "本系统"
	if knowledgeBase.ApplicationID != 0 {
		app, err := s.applicationService.Get(ctx, knowledgeBase.ApplicationID)
		if err != nil {
			logger.Error("获取应用失败", logger.F("err", err))
			return err
		}
		appName = app.Name
	}
	suffix := "公共知识库"
	if knowledgeBase.CustomID != "" {
		suffix = util.NewShortID()
	}
	if knowledgeBase.KnowledgeBaseName == "" {
		knowledgeBase.KnowledgeBaseName = fmt.Sprintf("%s知识库-%s", appName, suffix)
	} else if !strings.Contains(knowledgeBase.KnowledgeBaseName, "-") {
		knowledgeBase.KnowledgeBaseName += "-" + suffix
	}

	kbClient, err := s.GetDifyKnowledgeBaseClient(ctx)
	if err != nil {
		return err
	}

	id, err := kbClient.CreateKnowledgeBase(knowledgeBase.KnowledgeBaseName, appName)
	if err != nil {
		return err
	}
	knowledgeBase.OuterID = id

	if err := s.db.Create(knowledgeBase).Error; err != nil {
		logger.Error("创建知识库失败", logger.F("err", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *knowledgeBaseService) DeleteByApplicationID(ctx context.Context, applicationID uint64) error {
	if applicationID == 0 {
		return constant.ErrRecordIDEmpty
	}

	// 获取所有知识库
	var knowledgeBaseList []*model.KnowledgeBase
	if err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID}).Find(&knowledgeBaseList).Error; err != nil {
		logger.Error("获取知识库列表失败", logger.F("err", err))
		return constant.ErrDatabaseError
	}

	kbClient, err := s.GetDifyKnowledgeBaseClient(ctx)
	if err != nil {
		return err
	}

	// 循环删除每个知识库
	for _, knowledgeBase := range knowledgeBaseList {
		// 调用dify删除知识库
		if knowledgeBase.OuterID == "" {
			logger.Warn("知识库外部ID为空，忽略", logger.F("knowledgeBase", knowledgeBase))
			continue
		}
		err = kbClient.DeleteKnowledgeBase(knowledgeBase.OuterID)
		if err != nil {
			continue
		}
		// 删除知识库
		if err := s.Delete(ctx, knowledgeBase.ID); err != nil {
			logger.Error("删除知识库失败", logger.F("err", err))
			return constant.ErrDatabaseError
		}

	}
	return nil
}

func (s *knowledgeBaseService) GetByApplicationIDAndCustomID(ctx context.Context, applicationID uint64, customID string) (*model.KnowledgeBase, error) {
	var knowledgeBase model.KnowledgeBase
	err := s.db.
		Where("application_id = ? AND custom_id = ?", applicationID, customID).
		First(&knowledgeBase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error("获取知识库失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}
	return &knowledgeBase, nil
}
