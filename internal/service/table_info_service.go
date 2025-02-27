package service

import "github.com/yockii/dify_tools/internal/model"

type tableInfoService struct {
	*BaseServiceImpl[*model.TableInfo]
}

func NewTableInfoService() *tableInfoService {
	srv := new(tableInfoService)
	srv.BaseServiceImpl = NewBaseService[*model.TableInfo](BaseServiceConfig[*model.TableInfo]{
		NewModel: srv.NewModel,
	})
	return srv
}

func (s *tableInfoService) NewModel() *model.TableInfo {
	return &model.TableInfo{}
}
