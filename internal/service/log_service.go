package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"gorm.io/gorm"
)

type logService struct{}

func NewLogService() LogService {
	return &logService{}
}

func (s *logService) CreateLog(ctx context.Context, log *model.Log) error {
	if err := database.GetDB().Create(log).Error; err != nil {
		return fmt.Errorf("创建日志失败: %v", err)
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
		return nil, 0, fmt.Errorf("获取日志总数失败: %v", err)
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询日志失败: %v", err)
	}

	return logs, total, nil
}

func (s *logService) CreateLoginLog(ctx context.Context, userID uint64, ip, userAgent string, success bool) error {
	log := &model.Log{
		UserID:    userID,
		Action:    model.LogActionLogin,
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
		return fmt.Errorf("删除旧日志失败: %v", err)
	}
	return nil
}

func (s *logService) BatchCreateLogs(ctx context.Context, logs []*model.Log) error {
	if err := database.GetDB().CreateInBatches(logs, 100).Error; err != nil {
		return fmt.Errorf("批量创建日志失败: %v", err)
	}
	return nil
}

func (s *logService) GetUserLoginHistory(ctx context.Context, userID uint64, limit int) ([]*model.Log, error) {
	var logs []*model.Log
	if err := database.GetDB().Where("user_id = ? AND action = ?", userID, model.LogActionLogin).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询登录历史失败: %v", err)
	}
	return logs, nil
}

func (s *logService) GetUserLastLogin(ctx context.Context, userID uint64) (*model.Log, error) {
	var log model.Log
	if err := database.GetDB().Where("user_id = ? AND action = ? AND failed = false", userID, model.LogActionLogin).
		Order("created_at DESC").
		First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询上次登录信息失败: %v", err)
	}
	return &log, nil
}
