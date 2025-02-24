package service

import (
	"context"

	"github.com/yockii/dify_tools/internal/model"
	"gorm.io/gorm"
)

type dictService struct {
	*BaseService[*model.Dict]
	dictMap map[uint64]*model.Dict
}

func NewDictService() *dictService {
	return &dictService{
		NewBaseService[*model.Dict](),
		make(map[uint64]*model.Dict),
	}
}

func (s *dictService) NewModel() *model.Dict {
	return &model.Dict{}
}

func (s *dictService) CheckDuplicate(record *model.Dict) (bool, error) {
	query := s.db.Model(s.NewModel()).Where(&model.Dict{
		ParentID: record.ParentID,
		Code:     record.Code,
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
	dict, ok := s.dictMap[id]
	return dict, ok
}

func (s *dictService) CacheHook(ctx context.Context, dict *model.Dict) {
	s.dictMap[dict.ID] = dict
}

func (s *dictService) DeleteHook(ctx context.Context, dict *model.Dict) {
	delete(s.dictMap, dict.ID)
}

func (s *dictService) UpdateHook(ctx context.Context, dict *model.Dict) {
	delete(s.dictMap, dict.ID)
}
