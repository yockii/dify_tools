package service

import (
	"context"
	"errors"
	"mime/multipart"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type documentService struct {
	*BaseServiceImpl[*model.Document]
	knowledgeBaseService KnowledgeBaseService
	applicationService   ApplicationService
	dictService          DictService
}

func NewDocumentService(dictService DictService, applicationService ApplicationService, knowledgeBaseService KnowledgeBaseService) *documentService {
	srv := new(documentService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.Document]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
	})
	srv.applicationService = applicationService
	srv.dictService = dictService
	srv.knowledgeBaseService = knowledgeBaseService
	return srv
}

func (s *documentService) NewModel() *model.Document {
	return &model.Document{}
}

func (s *documentService) CheckDuplicate(record *model.Document) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.Document{
		FileName: record.FileName,
	})
	query = query.Where("application_id = ? AND custom_id = ?", record.ApplicationID, record.CustomID)

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

func (s *documentService) DeleteCheck(record *model.Document) error {
	return nil
}

func (s *documentService) BuildCondition(query *gorm.DB, condition *model.Document) *gorm.DB {
	query = query.Where("application_id = ?", condition.ApplicationID)
	if condition.CustomID != "" {
		query = query.Where("custom_id IN ('', ?)", condition.CustomID)
	} else {
		query = query.Where("custom_id = 0")
	}
	if condition.KnowledgeBaseID != 0 {
		query = query.Where("knowledge_base_id = ?", condition.KnowledgeBaseID)
	}
	if condition.FileName != "" {
		query = query.Where("file_name LIKE ?", "%"+condition.FileName+"%")
	}
	if condition.OuterID != "" {
		query = query.Where("outer_id = ?", condition.OuterID)
	}
	if condition.Batch != "" {
		query = query.Where("batch = ?", condition.Batch)
	}
	if condition.Status != 0 {
		query = query.Where("status = ?", condition.Status)
	}
	return query
}

func (s *documentService) GetDocument(ctx context.Context, condition *model.Document) (*model.Document, error) {
	var document model.Document
	if err := s.db.Where(condition).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error("查询文档失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return &document, nil
}

func (s *documentService) AddDocument(ctx context.Context, document *model.Document, fileHeader *multipart.FileHeader) (*model.KnowledgeBase, error) {
	if document.ApplicationID == 0 && document.CustomID == "" {
		return nil, constant.ErrInvalidParams
	}

	document.FileName = fileHeader.Filename
	document.FileSize = fileHeader.Size
	if duplicated, err := s.CheckDuplicate(document); err != nil {
		return nil, err
	} else if duplicated {
		return nil, constant.ErrRecordDuplicate
	}

	kb, err := s.knowledgeBaseService.GetByApplicationIDAndCustomID(ctx, document.ApplicationID, document.CustomID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		// 需要创建知识库
		kb = &model.KnowledgeBase{
			ApplicationID: document.ApplicationID,
			CustomID:      document.CustomID,
		}
		err = s.knowledgeBaseService.Create(ctx, kb)
		if err != nil {
			return nil, err
		}
	}

	document.KnowledgeBaseID = kb.ID

	kbClient, err := s.knowledgeBaseService.GetDifyKnowledgeBaseClient(ctx)
	if err != nil {
		return nil, err
	}

	// 上传文件
	resp, err := kbClient.CreateDocumentByFile(kb.OuterID, fileHeader, nil)
	if err != nil {
		return nil, err
	}
	respJson := gjson.Parse(resp)
	document.OuterID = respJson.Get("document.id").String()
	document.Batch = respJson.Get("batch").String()
	status := respJson.Get("document.display_status").String()
	document.Status = s.transferDocumentStatus(status)

	if err := s.Create(ctx, document); err != nil {
		logger.Error("创建文档失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}

	go s.RefreshDocumentStatusUntil(kb.ID, kb.OuterID, document.Batch, document.Status, 5)

	return kb, nil
}

// 异步处理
func (s *documentService) RefreshDocumentStatusUntil(knowledgeBaseID uint64, datasetID, batch string, currentStatus, utilStatus int) {
	if currentStatus == utilStatus {
		return
	}
	kbClient, err := s.knowledgeBaseService.GetDifyKnowledgeBaseClient(context.Background())
	if err != nil {
		logger.Error("获取dify客户端失败", logger.F("err", err))
		return
	}
	resp, err := kbClient.DocumentBatchIndexingStatus(datasetID, batch)
	if err != nil {
		logger.Error("获取文档状态失败", logger.F("err", err))
		// 等待一段时间后重试
		time.Sleep(time.Second * 15)
		s.RefreshDocumentStatusUntil(knowledgeBaseID, datasetID, batch, currentStatus, utilStatus)
		return
	}
	status := s.transferDocumentStatus(resp)
	// 更新数据库
	if err = s.db.Where(map[string]interface{}{
		"knowledge_base_id": knowledgeBaseID,
		"batch":             batch,
	}).Updates(&model.Document{
		Status: status,
	}).Error; err != nil {
		logger.Error("更新文档状态失败", logger.F("err", err))
	}
	if status != utilStatus {
		time.Sleep(time.Second * 15)
		s.RefreshDocumentStatusUntil(knowledgeBaseID, datasetID, batch, status, utilStatus)
	}
}

func (s *documentService) transferDocumentStatus(status string) int {
	switch status {
	case "queuing", "waiting":
		return 1
	case "paused":
		return 2
	case "parsing", "cleaning", "splitting", "indexing":
		return 3
	case "error":
		return 4
	case "available":
		return 5
	case "disabled":
		return 6
	case "archived":
		return 7
	default:
		return 5
	}
}

func (s *documentService) Delete(ctx context.Context, id uint64) error {
	document, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if document == nil {
		return constant.ErrRecordNotFound
	}

	knowledgeBase, err := s.knowledgeBaseService.Get(ctx, document.KnowledgeBaseID)
	if err != nil {
		return err
	}
	if knowledgeBase == nil {
		return constant.ErrRecordNotFound
	}

	kbClient, err := s.knowledgeBaseService.GetDifyKnowledgeBaseClient(ctx)
	if err != nil {
		return err
	}

	if knowledgeBase.OuterID != "" && document.OuterID != "" {
		err = kbClient.DeleteDocument(knowledgeBase.OuterID, document.OuterID)
		if err != nil {
			return err
		}
	}

	if err = s.BaseServiceImpl.Delete(ctx, id); err != nil {
		logger.Error("删除文档失败", logger.F("err", err))
		return constant.ErrDatabaseError
	}
	return nil
}
