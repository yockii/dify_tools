package service

import "github.com/yockii/dify_tools/internal/model"

type tableInfoService struct {
	*BaseService[*model.TableInfo]
}

func NewTableInfoService() *tableInfoService {
	return &tableInfoService{
		NewBaseService[*model.TableInfo](),
	}
}
