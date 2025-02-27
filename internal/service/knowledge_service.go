package service

import (
	"context"
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
	srv.BaseServiceImpl = NewBaseService[*model.KnowledgeBase](BaseServiceConfig[*model.KnowledgeBase]{
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
	if record.ID != 0 {
		query = query.Where("id <> ?", record.ID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
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
	return &knowledge, err
}

func (s *knowledgeBaseService) GetByApplicationID(applicationID uint64) ([]*model.KnowledgeBase, error) {
	var list []*model.KnowledgeBase
	err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID}).Find(&list).Error
	return list, err
}

func (s *knowledgeBaseService) GetByApplicationIDAndKnowledgeName(applicationID uint64, knowledgeName string) (*model.KnowledgeBase, error) {
	var knowledge model.KnowledgeBase
	err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID, KnowledgeBaseName: knowledgeName}).First(&knowledge).Error
	return &knowledge, err
}

func (s *knowledgeBaseService) Create(ctx context.Context, knowledgeBase *model.KnowledgeBase) error {
	if knowledgeBase.ApplicationID == 0 {
		return fmt.Errorf("应用ID不能为空")
	}
	app, err := s.applicationService.Get(ctx, knowledgeBase.ApplicationID)
	if err != nil {
		logger.Error("获取应用失败", logger.F("err", err))
		return err
	}
	if knowledgeBase.KnowledgeBaseName == "" {
		knowledgeBase.KnowledgeBaseName = fmt.Sprintf("%s知识库-%s", app.Name, util.NewShortID())
	} else if !strings.Contains(knowledgeBase.KnowledgeBaseName, "-") {
		knowledgeBase.KnowledgeBaseName += "-" + util.NewShortID()
	}

	// 调用dify接口创建知识库
	difyBaseUrlDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyBaseUrl)
	if err != nil {
		logger.Error("获取dify接口地址失败", logger.F("err", err))
		return err
	}
	if difyBaseUrlDict == nil || difyBaseUrlDict.Value == "" {
		logger.Error("未配置dify接口地址")
		return fmt.Errorf("未配置dify接口地址")
	}
	difyBaseUrl := difyBaseUrlDict.Value
	difyDatasetsTokenDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyDatasetsToken)
	if err != nil {
		logger.Error("获取dify知识库API密钥失败", logger.F("err", err))
		return err
	}
	if difyDatasetsTokenDict == nil || difyDatasetsTokenDict.Value == "" {
		logger.Error("未配置dify知识库API密钥")
		return fmt.Errorf("未配置dify知识库API密钥")
	}
	difyDatasetsToken := difyDatasetsTokenDict.Value

	id, err := dify.NewKnowLedgeBaseClient(difyBaseUrl, difyDatasetsToken).CreateKnowledgeBase(knowledgeBase.KnowledgeBaseName, app.Name)
	if err != nil {
		logger.Error("创建知识库失败", logger.F("err", err))
		return err
	}
	knowledgeBase.OuterID = id

	if err := s.Create(ctx, knowledgeBase); err != nil {
		logger.Error("创建知识库失败", logger.F("err", err))
		return err
	}
	return nil
}

func (s *knowledgeBaseService) DeleteByApplicationID(ctx context.Context, applicationID uint64) error {
	if applicationID == 0 {
		return fmt.Errorf("应用ID不能为空")
	}

	// 获取所有知识库
	var knowledgeBaseList []*model.KnowledgeBase
	if err := s.db.Where(&model.KnowledgeBase{ApplicationID: applicationID}).Find(&knowledgeBaseList).Error; err != nil {
		logger.Error("获取知识库列表失败", logger.F("err", err))
		return err
	}

	difyBaseUrlDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyBaseUrl)
	if err != nil {
		logger.Error("获取dify接口地址失败", logger.F("err", err))
		return err
	}
	if difyBaseUrlDict == nil || difyBaseUrlDict.Value == "" {
		logger.Error("未配置dify接口地址")
		return fmt.Errorf("未配置dify接口地址")
	}
	difyBaseUrl := difyBaseUrlDict.Value
	difyDatasetsTokenDict, err := s.dictService.GetByCode(ctx, constant.DictCodeDifyDatasetsToken)
	if err != nil {
		logger.Error("获取dify知识库API密钥失败", logger.F("err", err))
		return err
	}
	if difyDatasetsTokenDict == nil || difyDatasetsTokenDict.Value == "" {
		logger.Error("未配置dify知识库API密钥")
		return fmt.Errorf("未配置dify知识库API密钥")
	}
	difyDatasetsToken := difyDatasetsTokenDict.Value

	difyKnowledgeBaseClient := dify.NewKnowLedgeBaseClient(difyBaseUrl, difyDatasetsToken)
	// 循环删除每个知识库
	for _, knowledgeBase := range knowledgeBaseList {
		// 调用dify删除知识库
		if knowledgeBase.OuterID == "" {
			logger.Error("知识库外部ID为空，忽略", logger.F("knowledgeBase", knowledgeBase))
			continue
		}
		err = difyKnowledgeBaseClient.DeleteKnowledgeBase(knowledgeBase.OuterID)
		if err != nil {
			logger.Error("删除知识库失败", logger.F("err", err))
			continue
		}
		// 删除知识库
		if err := s.Delete(ctx, knowledgeBase.ID); err != nil {
			logger.Error("删除知识库失败", logger.F("err", err))
			return err
		}

	}
	return nil
}
