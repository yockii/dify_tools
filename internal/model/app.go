package model

import (
	"time"

	"gorm.io/gorm"

	"github.com/yockii/dify_tools/pkg/util"
)

// Application 应用模型
type Application struct {
	BaseModel
	Name              string    `json:"name" gorm:"type:varchar(50);not null"`
	Description       string    `json:"description" gorm:"type:varchar(200)"`
	APIKey            string    `json:"apiKey" gorm:"type:varchar(64);uniqueIndex"`
	Status            int       `json:"status" gorm:"type:int;default:1;not null"` // 1: 正常, -1: 禁用
	RateLimitInMinute int       `json:"rateLimitInMinute" gorm:"default:-1"`       // 每分钟限流, -1表示不限制
	AllowedOrigins    string    `json:"allowedOrigins"`                            // 允许的来源域名, 逗号分隔
	UpdatedAt         time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (a *Application) TableComment() string {
	return "应用表"
}

// BeforeCreate 创建前钩子
func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = util.NewID()
	}
	return nil
}

// DataSource 数据源模型
type DataSource struct {
	BaseModel
	ApplicationID uint64    `json:"applicationId,string" gorm:"index;not null"`
	Name          string    `json:"name" gorm:"type:varchar(50);not null"`
	Type          string    `json:"type" gorm:"type:varchar(20);not null"` // mysql, postgres, etc.
	Host          string    `json:"host" gorm:"type:varchar(50);not null"`
	Port          int       `json:"port" gorm:"type:int;not null"`
	User          string    `json:"user" gorm:"type:varchar(50);not null"`
	Password      string    `json:"password" gorm:"type:varchar(50);not null"`
	Database      string    `json:"database" gorm:"type:varchar(50);not null"`
	Schema        string    `json:"schema" gorm:"type:varchar(50);default:public"` // 数据库schema
	SyncTime      time.Time `json:"syncTime,omitzero" gorm:"type:timestamp"`
	Status        int       `json:"status" gorm:"type:int;default:1;not null"` // 1: 正常, -1: 禁用
	UpdatedAt     time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (d *DataSource) TableComment() string {
	return "数据源表"
}

// BeforeCreate 创建前钩子
func (d *DataSource) BeforeCreate(tx *gorm.DB) error {
	if d.ID == 0 {
		d.ID = util.NewID()
	}
	return nil
}

// TableInfo 表信息
type TableInfo struct {
	BaseModel
	ApplicationID uint64    `json:"applicationId,string" gorm:"index;not null"`
	DataSourceID  uint64    `json:"dataSourceId,string" gorm:"index;not null"`
	Name          string    `json:"name" gorm:"type:varchar(50);not null"`
	Comment       string    `json:"comment" gorm:"type:varchar(200)"`
	UpdatedAt     time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (t *TableInfo) TableComment() string {
	return "表信息表"
}

// BeforeCreate 创建前钩子
func (t *TableInfo) BeforeCreate(tx *gorm.DB) error {
	if t.ID == 0 {
		t.ID = util.NewID()
	}
	return nil
}

// ColumnInfo 列信息
type ColumnInfo struct {
	BaseModel
	ApplicationID uint64    `json:"applicationId,string" gorm:"index;not null"`
	DataSourceID  uint64    `json:"dataSourceId,string" gorm:"index;not null"`
	TableID       uint64    `json:"tableId,string" gorm:"index;not null"`
	Name          string    `json:"name" gorm:"type:varchar(50);not null"`
	Type          string    `json:"type" gorm:"type:varchar(50);not null"`
	Size          int64     `json:"size,string"`
	Precision     int64     `json:"precision"`
	Scale         int64     `json:"scale"`
	Nullable      bool      `json:"nullable"`
	DefaultValue  string    `json:"defaultValue"`
	Comment       string    `json:"comment"`
	UpdatedAt     time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (c *ColumnInfo) TableComment() string {
	return "列信息表"
}

// BeforeCreate 创建前钩子
func (c *ColumnInfo) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = util.NewID()
	}
	return nil
}

// Usage 使用统计
type Usage struct {
	BaseModel
	ApplicationID    uint64    `json:"applicationId,string" gorm:"index;not null"`
	Date             string    `json:"date" gorm:"type:varchar(10);index;not null"` // YYYY-MM-DD
	PromptTokens     int       `json:"promptTokens" gorm:"type:int;default:0;not null"`
	CompletionTokens int       `json:"completionTokens" gorm:"type:int;default:0;not null"`
	TotalTokens      int       `json:"totalTokens" gorm:"type:int;default:0;not null"`
	UpdatedAt        time.Time `json:"updatedAt,omitzero" gorm:"type:timestamp;not null"`
}

func (u *Usage) TableComment() string {
	return "使用统计表"
}

// BeforeCreate 创建前钩子
func (u *Usage) BeforeCreate(tx *gorm.DB) error {
	if u.ID == 0 {
		u.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &Application{}, &DataSource{}, &TableInfo{}, &ColumnInfo{}, &Usage{})
}
