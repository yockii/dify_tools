package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/yockii/dify_tools/pkg/util"
)

// User 用户模型
type User struct {
	BaseModel
	Username  string    `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	Password  string    `json:"password,omitempty" gorm:"type:varchar(100);not null"`
	RoleID    uint64    `json:"roleId,string" gorm:"index;not null"`
	Status    int       `json:"status" gorm:"type:int;default:1;not null"` // 1: 正常, -1: 禁用
	LastLogin time.Time `json:"lastLoginAt,omitzero" gorm:"type:timestamp"`
	UpdatedAt time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (u *User) TableComment() string {
	return "用户表"
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == 0 {
		u.ID = util.NewID()
	}
	if err := u.encryptPassword(); err != nil {
		return err
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") {

		pwd := ""
		switch dest := tx.Statement.Dest.(type) {
		case *User:
			pwd = dest.Password
		case map[string]interface{}:
			if pwdInter, ok := dest["Password"]; ok {
				pwd, _ = pwdInter.(string)
			}
		}
		if pwd != "" {
			if encryptPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost); err == nil {
				tx.Statement.SetColumn("Password", encryptPassword)
			}
		}

	}
	return nil
}

// ComparePassword 比较密码
func (u *User) ComparePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// encryptPassword 加密密码
func (u *User) encryptPassword() error {
	if u.Password == "" {
		return nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// Role 角色模型
type Role struct {
	BaseModel
	Name      string    `json:"name" gorm:"type:varchar(50);not null"`
	Code      string    `json:"code" gorm:"type:varchar(50);uniqueIndex;not null"`
	Status    int       `json:"status" gorm:"type:int;default:1;not null"` // 1: 正常, -1: 禁用
	UpdatedAt time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (r *Role) TableComment() string {
	return "角色表"
}

// BeforeCreate 创建前钩子
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == 0 {
		r.ID = util.NewID()
	}
	return nil
}

// Permission 权限资源模型
type Permission struct {
	BaseModel
	Code      string    `json:"code" gorm:"type:varchar(50);uniqueIndex;not null"`
	Name      string    `json:"name" gorm:"type:varchar(50);not null"`
	Type      string    `json:"type" gorm:"type:varchar(20);not null"`     // menu, button, api
	Status    int       `json:"status" gorm:"type:int;default:1;not null"` // 1: 正常, -1: 禁用
	ParentID  uint64    `json:"parentId,string" gorm:"index"`
	Sort      int       `json:"sort" gorm:"type:int;default:0"`
	UpdatedAt time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (p *Permission) TableComment() string {
	return "权限资源表"
}

// BeforeCreate 创建前钩子
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = util.NewID()
	}
	return nil
}

// RolePermission 角色-权限关联表
type RolePermission struct {
	BaseModel
	RoleID       uint64 `json:"roleId,string" gorm:"primaryKey"`
	PermissionID uint64 `json:"permissionId,string" gorm:"primaryKey"`
}

func (rp *RolePermission) TableComment() string {
	return "角色权限关联表"
}

// BeforeCreate 创建前钩子
func (rp *RolePermission) BeforeCreate(tx *gorm.DB) error {
	if rp.ID == 0 {
		rp.ID = util.NewID()
	}
	return nil
}

// Log 用户日志模型
type Log struct {
	BaseModel
	UserID    uint64 `json:"userId,string" gorm:"index;not null"`
	Action    int    `json:"action" gorm:"not null"`
	IP        string `json:"ip" gorm:"type:varchar(50)"`
	UserAgent string `json:"userAgent" gorm:"type:varchar(255)"`
	Failed    bool   `json:"failed" gorm:"default:false;not null"`
	User      *User  `json:"user" gorm:"foreignKey:UserID"` // 关联字段
}

func (l *Log) TableComment() string {
	return "日志表"
}

// BeforeCreate 创建前钩子
func (l *Log) BeforeCreate(tx *gorm.DB) error {
	if l.ID == 0 {
		l.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &User{}, &Role{}, &Permission{}, &RolePermission{}, &Log{})
}
