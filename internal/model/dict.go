package model

import (
	"strconv"

	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/util"
	"gorm.io/gorm"
)

type Dict struct {
	BaseModel
	Name     string `json:"name" gorm:"type:varchar(50);not null"`
	Code     string `json:"code" gorm:"type:varchar(50);not null"`
	Value    string `json:"value" gorm:"type:varchar(255);not null"`
	ParentID uint64 `json:"parentId,string" gorm:"not null"`
	Sort     int    `json:"sort" gorm:"type:int;default:0"`
}

func (d *Dict) ValueUint64() uint64 {
	v, err := strconv.ParseUint(d.Value, 10, 64)
	if err != nil {
		logger.Error("转换字典值到uint64失败", logger.F("value", d.Value), logger.F("err", err))
		return 0
	}
	return v
}

func (d *Dict) TableComment() string {
	return "字典表"
}

// BeforeCreate 创建前钩子
func (d *Dict) BeforeCreate(tx *gorm.DB) error {
	if d.ID == 0 {
		d.ID = util.NewID()
	}
	return nil
}

func init() {
	models = append(models, &Dict{})
}
