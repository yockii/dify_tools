package service

import (
	"context"
	"errors"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type usageService struct {
	*BaseServiceImpl[*model.Usage]
	usageMap map[uint64]string
}

func NewUsageService() *usageService {
	srv := new(usageService)
	srv.BaseServiceImpl = NewBaseService(BaseServiceConfig[*model.Usage]{
		NewModel: srv.NewModel,
		BuildCondition: func(query *gorm.DB, condition *model.Usage) *gorm.DB {
			if condition.ApplicationID > 0 {
				query = query.Where("application_id = ?", condition.ApplicationID)
			}
			if condition.Date != "" {
				query = query.Where("date = ?", condition.Date)
			}
			return query
		},
	})
	srv.usageMap = make(map[uint64]string)
	return srv
}

func (s *usageService) NewModel() *model.Usage {
	return &model.Usage{}
}

// ///// 覆盖不允许处理的方法
func (s *usageService) Update(ctx context.Context, record *model.Usage) error {
	return constant.ErrMethodNotAllow
}

func (s *usageService) Delete(ctx context.Context, id uint64) error {
	return constant.ErrMethodNotAllow
}

func (s *usageService) CreateByEndMessage(applicationID, agentID uint64, endMessage string) {
	j := gjson.Parse(endMessage)
	if j.Get("event").String() != "message_end" {
		return
	}

	now := time.Now()
	record := &model.Usage{
		ApplicationID:    applicationID,
		AgentID:          agentID,
		Date:             now.Format("2006-01-02"),
		PromptTokens:     int(j.Get("metadata.usage.prompt_tokens").Int()),
		CompletionTokens: int(j.Get("metadata.usage.completion_tokens").Int()),
		TotalTokens:      int(j.Get("metadata.usage.total_tokens").Int()),
	}
	err := s.Create(context.Background(), record)
	if err != nil {
		logger.Error("创建使用记录失败", logger.F("error", err), logger.F("record", record))
	}
}

func (s *usageService) Create(ctx context.Context, record *model.Usage) error {
	useIncrement := false
	if d, has := s.usageMap[record.ApplicationID]; has {
		useIncrement = d != record.Date
	}

	var existsUsage model.Usage

	if !useIncrement {
		// 检查数据库里是否存在
		query := s.db.Model(s.NewModel()).Where(map[string]interface{}{
			"application_id": record.ApplicationID,
			"date":           record.Date,
		})
		if err := query.First(&existsUsage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = s.db.Create(record).Error
				if err != nil {
					logger.Error("创建记录失败", logger.F("error", err))
					return constant.ErrDatabaseError
				}
				s.usageMap[record.ApplicationID] = record.Date
				return nil
			} else {
				logger.Error("查询记录失败", logger.F("error", err))
				return constant.ErrDatabaseError
			}
		}
	}

	// 增加token使用信息, 用gorm.Expr
	if err := s.db.Model(&existsUsage).
		Updates(map[string]interface{}{
			"PromptTokens":     gorm.Expr("prompt_tokens + ?", record.PromptTokens),
			"CompletionTokens": gorm.Expr("completion_tokens + ?", record.CompletionTokens),
			"TotalTokens":      gorm.Expr("total_tokens + ?", record.TotalTokens),
		}).Error; err != nil {
		logger.Error("更新记录失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}

	s.usageMap[record.ApplicationID] = record.Date
	return nil
}
