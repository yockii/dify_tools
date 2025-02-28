package service

import (
	"context"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type roleService struct {
	*BaseServiceImpl[*model.Role]
}

func NewRoleService() RoleService {
	srv := new(roleService)
	srv.BaseServiceImpl = NewBaseService[*model.Role](BaseServiceConfig[*model.Role]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		DeleteCheck:    srv.DeleteCheck,
		BuildCondition: srv.BuildCondition,
	})
	return srv
}

func (s *roleService) NewModel() *model.Role {
	return &model.Role{}
}

func (s *roleService) CheckDuplicate(record *model.Role) (bool, error) {
	query := s.db.Model(s.NewModel()).Where("code = ?", record.Code)
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

func (s *roleService) DeleteCheck(record *model.Role) error {
	// 检查是否有用户使用该角色
	var count int64
	if err := database.GetDB().Model(&model.User{}).Where("role_id = ?", record.ID).Count(&count).Error; err != nil {
		logger.Error("查询角色是否使用失败", logger.F("error", err))
		return constant.ErrDatabaseError
	}
	if count > 0 {
		return constant.ErrRoleInUse
	}
	return nil
}

func (s *roleService) GetRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := s.db.First(&role, "code = ?", code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constant.ErrRecordNotFound
		}
		logger.Error("查询角色失败", logger.F("error", err))
		return nil, constant.ErrDatabaseError
	}

	return &role, nil
}

func (s *roleService) BuildCondition(query *gorm.DB, condition *model.Role) *gorm.DB {
	if condition != nil {
		if condition.ID != 0 {
			query = query.Where("id = ?", condition.ID)
		}
		if condition.Code != "" {
			query = query.Where("code like ?", "%"+condition.Code+"%")
		}
		if condition.Status != 0 {
			query = query.Where("status = ?", condition.Status)
		}
	}
	return query
}
