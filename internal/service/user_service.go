package service

import (
	"context"
	"fmt"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/logger"
	"gorm.io/gorm"
)

type userService struct {
	*BaseServiceImpl[*model.User]
}

func NewUserService() *userService {
	srv := new(userService)
	srv.BaseServiceImpl = NewBaseService[*model.User](BaseServiceConfig[*model.User]{
		NewModel:       srv.NewModel,
		CheckDuplicate: srv.CheckDuplicate,
		BuildCondition: srv.BuildCondition,
		ListOrder:      srv.ListOrder,
	})
	return srv
}

func (s *userService) NewModel() *model.User {
	return &model.User{}
}

func (s *userService) CheckDuplicate(record *model.User) (bool, error) {
	query := s.db.Model(s.NewModel()).Where("username = ?", record.Username)
	if record.ID != 0 {
		query = query.Where("id <> ?", record.ID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		logger.Error("查询记录失败", logger.F("error", err))
		return false, fmt.Errorf("检查用户名失败: %v", err)
	}
	return count > 0, nil
}

func (s *userService) BuildCondition(query *gorm.DB, condition *model.User) *gorm.DB {
	if condition != nil {
		if condition.ID != 0 {
			query = query.Where("id = ?", condition.ID)
		}
		if condition.Username != "" {
			query = query.Where("username like ?", "%"+condition.Username+"%")
		}
	}
	return query
}

func (s *userService) ListOrder() string {
	return "created_at DESC"
}

func (s *userService) UpdateUser(ctx context.Context, user *model.User) error {
	// 检查用户是否存在
	var existingUser model.User
	if err := s.db.First(&existingUser, "id = ?", user.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户未找到")
		}
		logger.Error("查询用户失败", logger.F("error", err))
		return fmt.Errorf("查询用户失败: %v", err)
	}

	user.Password = ""

	// 更新用户信息
	if err := s.db.Model(user).Omit("Password", "Status", "LastLogin", "CreatedAt", "UpdatedAt").Updates(user).Error; err != nil {
		logger.Error("更新用户失败", logger.F("error", err))
		return fmt.Errorf("更新用户失败: %v", err)
	}

	return nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, "username = ?", username).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户未找到")
		}
		logger.Error("查询用户失败", logger.F("error", err))
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	return &user, nil
}

func (s *userService) UpdatePassword(ctx context.Context, id uint64, oldPassword, newPassword string) error {
	// 获取用户信息
	user, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// 验证旧密码
	if !user.ComparePassword(oldPassword) {
		return fmt.Errorf("旧密码不正确")
	}

	// 更新用户密码
	if err := s.db.Model(user).Updates(&model.User{
		Password: newPassword,
	}).Error; err != nil {
		logger.Error("更新用户密码失败", logger.F("error", err))
		return fmt.Errorf("更新用户密码失败: %v", err)
	}

	return nil
}

func (s *userService) UpdateStatus(ctx context.Context, id uint64, status int) error {
	// 获取用户信息
	user, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// 更新状态
	if err := s.db.Model(user).Updates(&model.User{
		Status: status,
	}).Error; err != nil {
		logger.Error("更新用户状态失败", logger.F("error", err))
		return err
	}

	return nil
}
