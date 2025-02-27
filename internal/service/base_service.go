package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type BaseServiceConfig[T model.Model] struct {
	NewModel        func() T
	CheckDuplicate  func(record T) (bool, error)
	DeleteCheck     func(record T) error
	BuildCondition  func(query *gorm.DB, condition T) *gorm.DB
	ListOrder       func() string
	ListOmitColumns func() []string
	UpdateHook      func(ctx context.Context, record T)
	DeleteHook      func(ctx context.Context, record T)
	CacheHook       func(ctx context.Context, record T)
	GetFromCache    func(ctx context.Context, id uint64) (T, bool)
}

type BaseServiceImpl[T model.Model] struct {
	db     *gorm.DB
	config BaseServiceConfig[T]
}

func NewBaseService[T model.Model](
	config BaseServiceConfig[T],
) *BaseServiceImpl[T] {
	return &BaseServiceImpl[T]{
		db:     database.GetDB(),
		config: config,
	}
}

func (s *BaseServiceImpl[T]) BaseCheckDuplicate(record T) (bool, error) {
	if s.config.CheckDuplicate != nil {
		return s.config.CheckDuplicate(record)
	}
	return false, nil
}

func (s *BaseServiceImpl[T]) BaseNewModel() T {
	if s.config.NewModel != nil {
		return s.config.NewModel()
	}

	var t T
	tType := reflect.TypeOf(t)

	// 如果 T 是指针类型，则需要创建指针指向的对象
	if tType.Kind() == reflect.Ptr {
		tType = tType.Elem()                        // 获取指针指向的类型
		value := reflect.New(tType).Interface().(T) // 创建指针类型的实例
		return value
	}

	// 如果 T 是值类型，则直接创建实例
	return reflect.New(tType).Elem().Interface().(T)
}

func (s *BaseServiceImpl[T]) BaseListOrder() string {
	if s.config.ListOrder != nil {
		return s.config.ListOrder()
	}
	return "created_at DESC"
}

func (s *BaseServiceImpl[T]) BaseDeleteCheck(record T) error {
	if s.config.DeleteCheck != nil {
		return s.config.DeleteCheck(record)
	}
	return nil
}

func (s *BaseServiceImpl[T]) BaseDeleteHook(ctx context.Context, record T) {
	if s.config.DeleteHook != nil {
		s.config.DeleteHook(ctx, record)
	}
}

////////////////////////////////////////////////

// Create 创建记录
func (s *BaseServiceImpl[T]) Create(ctx context.Context, record T) error {
	// 检查是否重复
	if duplicate, err := s.BaseCheckDuplicate(record); err != nil {
		return err
	} else if duplicate {
		return fmt.Errorf("记录已存在")
	}

	if err := s.db.Create(record).Error; err != nil {
		logger.Error("创建记录失败", logger.F("error", err))
		return fmt.Errorf("创建记录失败: %v", err)
	}
	return nil
}

// Update 更新记录
func (s *BaseServiceImpl[T]) Update(ctx context.Context, record T) error {
	id := record.GetID()
	if id == 0 {
		return fmt.Errorf("ID不能为空")
	}
	// existingRecord := s.NewModel()
	var existingRecord T
	if err := s.db.First(&existingRecord, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("记录不存在")
		}
		logger.Error("查询记录失败", logger.F("error", err))
		return fmt.Errorf("查询记录失败: %v", err)
	}

	// 检查是否重复
	if duplicate, err := s.BaseCheckDuplicate(record); err != nil {
		return err
	} else if duplicate {
		return fmt.Errorf("记录已存在")
	}

	// 更新记录
	if err := s.db.Model(record).Updates(record).Error; err != nil {
		logger.Error("更新记录失败", logger.F("error", err))
		return fmt.Errorf("更新记录失败: %v", err)
	}

	if s.config.UpdateHook != nil {
		s.config.UpdateHook(ctx, existingRecord)
	}
	return nil
}

// Delete 删除记录
func (s *BaseServiceImpl[T]) Delete(ctx context.Context, id uint64) error {
	var record T
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("记录不存在")
		}
		logger.Error("查询记录失败", logger.F("error", err))
		return fmt.Errorf("查询记录失败: %v", err)
	}

	// 检查是否可以删除
	if err := s.BaseDeleteCheck(record); err != nil {
		return err
	}

	// 删除记录
	if err := s.db.Delete(record).Error; err != nil {
		logger.Error("删除记录失败", logger.F("error", err))
		return fmt.Errorf("删除记录失败: %v", err)
	}

	s.BaseDeleteHook(ctx, record)

	return nil
}

// Get 查询记录
func (s *BaseServiceImpl[T]) Get(ctx context.Context, id uint64) (T, error) {
	if s.config.GetFromCache != nil {
		if result, ok := s.config.GetFromCache(ctx, id); ok {
			return result, nil
		}
	}
	var record T
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return record, fmt.Errorf("记录不存在")
		}
		logger.Error("查询记录失败", logger.F("error", err))
		return record, fmt.Errorf("查询记录失败: %v", err)
	}
	if s.config.CacheHook != nil {
		s.config.CacheHook(ctx, record)
	}
	return record, nil
}

// List 查询记录列表
func (s *BaseServiceImpl[T]) List(ctx context.Context, condition T, offset, limit int) ([]T, int64, error) {
	var records []T
	var total int64

	query := s.db.Model(s.BaseNewModel())

	if s.config.ListOmitColumns != nil {
		if omits := s.config.ListOmitColumns(); len(omits) > 0 {
			query = query.Omit(omits...)
		}
	}

	// 构建查询条件
	if s.config.BuildCondition != nil {
		query = s.config.BuildCondition(query, condition)
	}
	// 查询记录总数
	if err := query.Count(&total).Error; err != nil {
		logger.Error("查询记录总数失败", logger.F("error", err))
		return records, 0, fmt.Errorf("查询记录总数失败: %v", err)
	}

	if total > 0 && limit > 0 {
		// 查询记录列表
		if err := query.Offset(offset).Limit(limit).Order(s.BaseListOrder()).Find(&records).Error; err != nil {
			logger.Error("查询记录失败", logger.F("error", err))
			return records, 0, fmt.Errorf("查询记录失败: %v", err)
		}
	}

	return records, total, nil
}
