package service

import (
	"context"
	"fmt"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/database"
	"gorm.io/gorm"
)

type roleService struct {
	*BaseService[*model.Role]
}

func NewRoleService() RoleService {
	return &roleService{
		NewBaseService[*model.Role](),
	}
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
		return false, fmt.Errorf("检查角色代码失败: %v", err)
	}
	return count > 0, nil
}

func (s *roleService) DeleteCheck(record *model.Role) error {
	// 检查是否有用户使用该角色
	var count int64
	if err := database.GetDB().Model(&model.User{}).Where("role_id = ?", record.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("查询角色是否使用失败: %v", err)
	}
	if count > 0 {
		return fmt.Errorf("角色正在使用，无法删除")
	}
	return nil
}

func (s *roleService) GetRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := s.db.First(&role, "code = ?", code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色不存在")
		}
		return nil, fmt.Errorf("查询角色失败: %v", err)
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
