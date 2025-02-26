package model

import (
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

type KnowledgeBase struct {
	BaseModel
	OuterID           string `json:"outerId" gorm:"type:varchar(50);not null;index"`
	ApplicationID     uint64 `json:"applicationId,string" gorm:"index;not null"`
	KnowledgeBaseName string `json:"knowledgeBaseName" gorm:"type:varchar(50);not null"`
}

func (k *KnowledgeBase) TableComment() string {
	return "知识库表"
}

func (k *KnowledgeBase) BeforeCreate(tx *gorm.DB) error {
	if k.ID == 0 {
		k.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &KnowledgeBase{})
}
