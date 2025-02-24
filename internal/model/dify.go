package model

import (
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

const (
	// AgentTypeEmbed 内嵌代理
	AgentTypeEmbed = 1 + iota
	// AgentTypeApplication 应用代理
	AgentTypeApplication
)

const (
	InnerAgentCodeSqlBuilder = "sql_builder"
)

const (
	InnerAgentNameSqlBuilder = "SQL构建器"
)

type Agent struct {
	BaseModel
	Code      string `json:"code" gorm:"type:varchar(100);not null;unique_index"`
	Name      string `json:"name" gorm:"type:varchar(100);not null"`
	Type      int    `json:"type" gorm:"type:int;not null"`
	ApiSecret string `json:"apiSecret" gorm:"type:varchar(100);not null"`
}

func (a *Agent) TableComment() string {
	return "AI代理配置表"
}

// BeforeCreate 创建前钩子
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = util.NewID()
	}
	return nil
}

type ApplicationAgent struct {
	BaseModel
	ApplicationID uint64 `json:"applicationId,string" gorm:"index;not null"`
	AgentID       uint64 `json:"agentId,string" gorm:"index;not null"`
}

func (a *ApplicationAgent) TableComment() string {
	return "应用代理关联表"
}

// BeforeCreate 创建前钩子
func (a *ApplicationAgent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &Agent{}, &ApplicationAgent{})
}
