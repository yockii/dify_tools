package service

import (
	"context"
	"errors"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type dictService struct {
	*BaseServiceImpl[*model.Dict]
	dictIDMap   map[uint64]*model.Dict
	dictCodeMap map[string]*model.Dict
}

func NewDictService() *dictService {
	srv := new(dictService)
	srv.BaseServiceImpl = NewBaseService[*model.Dict](BaseServiceConfig[*model.Dict]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
		ListOrder:      srv.ListOrder,
		GetFromCache:   srv.GetFromCache,
		CacheHook:      srv.CacheHook,
		DeleteHook:     srv.DeleteHook,
		UpdateHook:     srv.UpdateHook,
	})
	srv.dictIDMap = make(map[uint64]*model.Dict)
	srv.dictCodeMap = make(map[string]*model.Dict)
	return srv
}

func (s *dictService) NewModel() *model.Dict {
	return &model.Dict{}
}

func (s *dictService) CheckDuplicate(record *model.Dict) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.Dict{
		Code: record.Code,
	})
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

func (s *dictService) DeleteCheck(record *model.Dict) error {
	return nil
}

func (s *dictService) BuildCondition(query *gorm.DB, condition *model.Dict) *gorm.DB {
	query = query.Where("parent_id = ?", condition.ParentID)
	if condition.Code != "" {
		query = query.Where("code LIKE ?", "%"+condition.Code+"%")
	}
	if condition.Name != "" {
		query = query.Where("name LIKE ?", "%"+condition.Name+"%")
	}
	return query
}

func (s *dictService) ListOrder() string {
	return "sort"
}

func (s *dictService) GetFromCache(ctx context.Context, id uint64) (*model.Dict, bool) {
	dict, ok := s.dictIDMap[id]
	return dict, ok
}

func (s *dictService) CacheHook(ctx context.Context, dict *model.Dict) {
	s.dictIDMap[dict.ID] = dict
	s.dictCodeMap[dict.Code] = dict
}

func (s *dictService) DeleteHook(ctx context.Context, dict *model.Dict) {
	delete(s.dictIDMap, dict.ID)
	delete(s.dictCodeMap, dict.Code)
}

func (s *dictService) UpdateHook(ctx context.Context, dict *model.Dict) {
	delete(s.dictIDMap, dict.ID)
	delete(s.dictCodeMap, dict.Code)
}

func (s *dictService) GetByCode(ctx context.Context, code string) (*model.Dict, error) {
	if dict, ok := s.dictCodeMap[code]; ok {
		return dict, nil
	}
	// 从数据库中查询
	var dict model.Dict
	err := s.db.Where(&model.Dict{Code: code}).First(&dict).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrRecordNotFound
		}
		logger.Error("查询记录失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}
	s.CacheHook(ctx, &dict)
	return &dict, nil
}
