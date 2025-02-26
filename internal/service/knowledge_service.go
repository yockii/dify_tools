package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

type knowledgeBaseService struct {
	*BaseService[*model.KnowledgeBase]
	dictService DictService
}

func NewKnowledgeBaseService(dictService DictService) *knowledgeBaseService {
	return &knowledgeBaseService{
		NewBaseService[*model.KnowledgeBase](),
		dictService,
	}
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

func (s *knowledgeBaseService) CreateKnowledgeBase(ctx context.Context, knowledgeBase *model.KnowledgeBase) error {
	if knowledgeBase.ApplicationID == 0 {
		return fmt.Errorf("应用ID不能为空")
	}
	if knowledgeBase.KnowledgeBaseName == "" {
		knowledgeBase.KnowledgeBaseName = fmt.Sprintf("%s知识库-%s", knowledgeBase.KnowledgeBaseName, util.NewShortID())
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

	// 构建body的json
	body := map[string]string{
		"name":               knowledgeBase.KnowledgeBaseName,
		"description":        "",
		"indexing_technique": "high_quality",
		"permission":         "all_team_members",
		"provider":           "vendor",
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		logger.Error("构建dify接口请求body失败", logger.F("err", err))
		return err
	}

	httpClient := http.Client{}
	req, err := http.NewRequest("POST", difyBaseUrl+"/datasets", bytes.NewReader(bodyJson))
	if err != nil {
		logger.Error("构建dify接口请求失败", logger.F("err", err))
		return err
	}
	req.Header.Set("Authorization", "Bearer "+difyDatasetsToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("调用dify接口失败", logger.F("err", err))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("dify接口返回错误：%d", resp.StatusCode)
		logger.Error("调用dify接口失败", logger.F("err", err))
	}

	// 解析返回结果,只关注返回json中的id
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取dify接口返回body失败", logger.F("err", err))
		return err
	}
	response := gjson.ParseBytes(respBody)
	knowledgeBase.OuterID = response.Get("id").String()

	if err := s.Create(ctx, knowledgeBase); err != nil {
		logger.Error("创建知识库失败", logger.F("err", err))
		return err
	}
	return nil
}
