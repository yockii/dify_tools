package model

import (
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

type KnowledgeBase struct {
	BaseModel
	OuterID           string `json:"outerId" gorm:"type:varchar(50);not null;index"`
	ApplicationID     uint64 `json:"applicationId,string" gorm:"index;not null"`
	CustomID          string `json:"customId" gorm:"type:varchar(50);not null;index"`
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

type Document struct {
	BaseModel
	ApplicationID   uint64 `json:"applicationId,string" gorm:"index;not null"`
	KnowledgeBaseID uint64 `json:"knowledgeBaseId,string" gorm:"index;not null"`
	CustomID        string `json:"customId" gorm:"type:varchar(50);not null;index"`
	FileName        string `json:"fileName" gorm:"type:varchar(200);not null"`
	FileSize        int64  `json:"fileSize" gorm:"not null"`
	OuterID         string `json:"outerId" gorm:"type:varchar(50);not null;index"`
	Batch           string `json:"batch" gorm:"type:varchar(50);not null;index"`
	Status          int    `json:"status" gorm:"not null"`
}

func (d *Document) TableComment() string {
	return "文档表"
}

func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == 0 {
		d.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &KnowledgeBase{}, &Document{})
}
