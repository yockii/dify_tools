package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"gorm.io/gorm"
)

type BaseService[T model.Model] struct {
	db *gorm.DB
}

func NewBaseService[T model.Model]() *BaseService[T] {
	return &BaseService[T]{
		db: database.GetDB(),
	}
}

func (s *BaseService[T]) NewModel() T {
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

func (s *BaseService[T]) CheckDuplicate(record T) (bool, error) {
	return false, nil
}

func (s *BaseService[T]) DeleteCheck(record T) error {
	return nil
}

func (s *BaseService[T]) BuildCondition(query *gorm.DB, condition T) *gorm.DB {
	return query
}

func (s *BaseService[T]) ListOrder() string {
	return "created_at DESC"
}

func (s *BaseService[T]) ListOmitColumns() []string {
	return nil
}

// Create 创建记录
func (s *BaseService[T]) Create(ctx context.Context, record T) error {
	// 检查是否重复
	duplicate, err := s.CheckDuplicate(record)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("记录已存在")
	}

	if err := s.db.Create(record).Error; err != nil {
		return fmt.Errorf("创建记录失败: %v", err)
	}
	return nil
}

func (s *BaseService[T]) UpdateHook(ctx context.Context, record T) {
}

// Update 更新记录
func (s *BaseService[T]) Update(ctx context.Context, record T) error {
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
		return fmt.Errorf("查询记录失败: %v", err)
	}

	// 检查是否重复
	duplicate, err := s.CheckDuplicate(record)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("记录已存在")
	}

	// 更新记录
	if err := s.db.Model(record).Updates(record).Error; err != nil {
		return fmt.Errorf("更新记录失败: %v", err)
	}

	s.UpdateHook(ctx, existingRecord)
	return nil
}

func (s *BaseService[T]) DeleteHook(ctx context.Context, record T) {
}

// Delete 删除记录
func (s *BaseService[T]) Delete(ctx context.Context, id uint64) error {
	var record T
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("记录不存在")
		}
		return fmt.Errorf("查询记录失败: %v", err)
	}

	// 检查是否可以删除
	if err := s.DeleteCheck(record); err != nil {
		return err
	}

	// 删除记录
	if err := s.db.Delete(record).Error; err != nil {
		return fmt.Errorf("删除记录失败: %v", err)
	}
	s.DeleteHook(ctx, record)
	return nil
}

func (s *BaseService[T]) GetFromCache(ctx context.Context, id uint64) (T, bool) {
	return s.NewModel(), false
}

func (s *BaseService[T]) CacheHook(ctx context.Context, record T) {
}

// Get 查询记录
func (s *BaseService[T]) Get(ctx context.Context, id uint64) (T, error) {
	if result, ok := s.GetFromCache(ctx, id); ok {
		return result, nil
	}
	var record T
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return record, fmt.Errorf("记录不存在")
		}
		return record, fmt.Errorf("查询记录失败: %v", err)
	}
	s.CacheHook(ctx, record)
	return record, nil
}

// List 查询记录列表
func (s *BaseService[T]) List(ctx context.Context, condition T, offset, limit int) ([]T, int64, error) {
	var records []T
	var total int64

	query := s.db.Model(s.NewModel())

	if omits := s.ListOmitColumns(); len(omits) > 0 {
		query = query.Omit(omits...)
	}

	// 构建查询条件
	query = s.BuildCondition(query, condition)

	// 查询记录总数
	if err := query.Count(&total).Error; err != nil {
		return records, 0, fmt.Errorf("查询记录总数失败: %v", err)
	}

	// 查询记录列表
	if err := query.Offset(offset).Limit(limit).Order(s.ListOrder()).Find(&records).Error; err != nil {
		return records, 0, fmt.Errorf("查询记录失败: %v", err)
	}

	return records, total, nil
}
