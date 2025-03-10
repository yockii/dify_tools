package service

import (
	"context"
	"mime/multipart"

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
	switch status {
	case "queuing", "waiting":
		document.Status = 1
	case "paused":
		document.Status = 2
	case "parsing", "cleaning", "splitting", "indexing":
		document.Status = 3
	case "error":
		document.Status = 4
	case "available":
		document.Status = 5
	case "disabled":
		document.Status = 6
	case "archived":
		document.Status = 7
	default:
		document.Status = 5
	}

	if err := s.Create(ctx, document); err != nil {
		logger.Error("创建文档失败", logger.F("err", err))
		return nil, constant.ErrDatabaseError
	}

	return kb, nil
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
