package difyapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/datasource"
	"github.com/yockii/dify_tools/internal/middleware"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type DatabaseHandler struct {
	applicationService service.ApplicationService
	dataSourceService  service.DataSourceService
	tableInfoService   service.TableInfoService
	columnInfoService  service.ColumnInfoService
}

func RegisterDatabaseHandler(
	applicationService service.ApplicationService,
	dataSourceService service.DataSourceService,
	tableInfoService service.TableInfoService,
	columnInfoService service.ColumnInfoService,
) {
	handler := &DatabaseHandler{
		applicationService: applicationService,
		dataSourceService:  dataSourceService,
		tableInfoService:   tableInfoService,
		columnInfoService:  columnInfoService,
	}
	Handlers = append(Handlers, handler)
}

func (h *DatabaseHandler) RegisterRoutesV1_1(router fiber.Router) {
	h.RegisterRoutesV1(router)
}

func (h *DatabaseHandler) RegisterRoutesV1(router fiber.Router) {
	router.Get("/databases", middleware.NewAppMiddleware(h.applicationService), h.GetDatabases)
	router.Get("/schema", middleware.NewAppMiddleware(h.applicationService), h.GetDatabaseSchema)
	router.Post("/executeSql", middleware.NewAppMiddleware(h.applicationService), h.ExecuteSqlForDatabase)
}

func (h *DatabaseHandler) GetDatabases(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidCredential))
	}

	list, err := h.dataSourceService.ListForDify(c.Context(), &model.DataSource{
		ApplicationID: application.ID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(list))
}

func (h *DatabaseHandler) GetDatabaseSchema(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidCredential))
	}
	type Req struct {
		DatasourceID uint64 `json:"datasourceId,string"`
	}
	req := new(Req)
	if err := c.QueryParser(req); err != nil {
		logger.Error("请求参数解析失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	dataSource, err := h.dataSourceService.Get(c.Context(), req.DatasourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	if dataSource.ApplicationID != application.ID {
		return c.Status(fiber.StatusForbidden).JSON(service.Error(constant.ErrForbidden))
	}

	type TableWithColumns struct {
		*model.TableInfo
		Columns []*model.ColumnInfo `json:"columns"`
	}

	// 获取database同步好的数据记录
	var tables []*TableWithColumns
	var list []*model.TableInfo
	list, err = h.tableInfoService.ListSchemaForDify(c.Context(), &model.TableInfo{
		DataSourceID: dataSource.ID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	for _, t := range list {
		twc := new(TableWithColumns)
		twc.TableInfo = t
		var cl []*model.ColumnInfo

		cl, err = h.columnInfoService.ListSchemaForDify(c.Context(), &model.ColumnInfo{
			TableID: t.ID,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
		}
		twc.Columns = cl
		twc.ID = 0
		tables = append(tables, twc)
	}

	return c.JSON(service.OK(tables))
}

func (h *DatabaseHandler) ExecuteSqlForDatabase(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidCredential))
	}

	type Req struct {
		Sql          string `json:"sql"`
		DataSourceID uint64 `json:"datasourceId,string"`
	}
	req := new(Req)
	if err := c.BodyParser(req); err != nil {
		logger.Error("请求参数解析失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if req.Sql == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	// 检查datasource是否该应用
	dataSource, err := h.dataSourceService.Get(c.Context(), req.DataSourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	if dataSource.ApplicationID != application.ID {
		return c.Status(fiber.StatusForbidden).JSON(service.Error(constant.ErrForbidden))
	}

	// 得到datasource的连接
	db, err := datasource.GetDB(dataSource)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	// 进行查询，并把结果放到map[string]interface{}
	var result []map[string]interface{}
	err = db.Raw(req.Sql).Find(&result).Error
	if err != nil {
		logger.Error("执行sql失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	return c.JSON(service.OK(result))
}
