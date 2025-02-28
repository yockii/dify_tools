package service

import (
	"context"
	"time"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type logService struct{}

func NewLogService() LogService {
	return &logService{}
}

func (s *logService) CreateLog(ctx context.Context, log *model.Log) error {
	if err := database.GetDB().Create(log).Error; err != nil {
		logger.Error("创建日志失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *logService) ListLogs(ctx context.Context, userID uint64, actions []int, offset, limit int) ([]*model.Log, int64, error) {
	var logs []*model.Log
	var total int64

	query := database.GetDB().Model(&model.Log{})
	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}
	if len(actions) > 0 {
		query = query.Where("action IN ?", actions)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Error("获取日志总数失败", logger.F("error", err))
		return nil, 0, constant.ErrDatabaseError
	}

	if total > 0 && limit > 0 {
		// 获取列表
		if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Preload("User").Find(&logs).Error; err != nil {
			logger.Error("查询日志失败", logger.F("error", err))
			return nil, 0, constant.ErrDatabaseError
		}

		// 处理用户信息中的敏感字段
		for _, log := range logs {
			if log.User != nil {
				log.User.Password = ""
				log.User.RoleID = 0
				log.User.CreatedAt = time.Time{}
				log.User.UpdatedAt = time.Time{}
			}
		}
	}

	return logs, total, nil
}

func (s *logService) CreateLoginLog(ctx context.Context, userID uint64, ip, userAgent string, success bool) error {
	log := &model.Log{
		UserID:    userID,
		Action:    constant.LogActionLogin,
		IP:        ip,
		UserAgent: userAgent,
		Failed:    !success,
	}

	return s.CreateLog(ctx, log)
}

func (s *logService) CreateOperationLog(ctx context.Context, userID uint64, action int, ip, userAgent string) error {
	log := &model.Log{
		UserID:    userID,
		Action:    action,
		IP:        ip,
		UserAgent: userAgent,
	}

	return s.CreateLog(ctx, log)
}

func (s *logService) DeleteOldLogs(ctx context.Context, days int) error {
	deadline := time.Now().AddDate(0, 0, -days)
	if err := database.GetDB().Where("created_at < ?", deadline).Delete(&model.Log{}).Error; err != nil {
		logger.Error("删除旧日志失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *logService) BatchCreateLogs(ctx context.Context, logs []*model.Log) error {
	if err := database.GetDB().CreateInBatches(logs, 100).Error; err != nil {
		logger.Error("批量创建日志失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	return nil
}

func (s *logService) GetUserLoginHistory(ctx context.Context, userID uint64, limit int) ([]*model.Log, error) {
	var logs []*model.Log
	if err := database.GetDB().Where("user_id = ? AND action = ?", userID, constant.LogActionLogin).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		logger.Error("查询登录历史失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return logs, nil
}

func (s *logService) GetUserLastLogin(ctx context.Context, userID uint64) (*model.Log, error) {
	var log model.Log
	if err := database.GetDB().Where("user_id = ? AND action = ? AND failed = false", userID, constant.LogActionLogin).
		Order("created_at DESC").
		First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error("查询上次登录信息失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	return &log, nil
}
