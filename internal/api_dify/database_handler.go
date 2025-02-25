package difyapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/datasource"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type DatabaseHandler struct {
	dataSourceService service.DataSourceService
	tableInfoService  service.TableInfoService
	columnInfoService service.ColumnInfoService
}

func RegisterDatabaseHandler(
	dataSourceService service.DataSourceService,
	tableInfoService service.TableInfoService,
	columnInfoService service.ColumnInfoService,
) {
	handler := &DatabaseHandler{
		dataSourceService: dataSourceService,
		tableInfoService:  tableInfoService,
		columnInfoService: columnInfoService,
	}
	Handlers = append(Handlers, handler)
}

func (h *DatabaseHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/databases", h.GetDatabases)
	router.Get("/schema", h.GetDatabaseSchema)
	router.Post("/executeSql", h.ExecuteSqlForDatabase)
}

func (h *DatabaseHandler) GetDatabases(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidCredential))
	}

	list, _, err := h.dataSourceService.List(c.Context(), &model.DataSource{
		ApplicationID: application.ID,
	}, 0, 100)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(list))
}

func (h *DatabaseHandler) GetDatabaseSchema(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidCredential))
	}
	type Req struct {
		DatasourceID uint64 `json:"datasourceId,string"`
	}
	req := new(Req)
	if err := c.QueryParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidParams))
	}
	dataSource, err := h.dataSourceService.Get(c.Context(), req.DatasourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
	}
	if dataSource.ApplicationID != application.ID {
		return c.Status(fiber.StatusForbidden).JSON(service.Error(service.ErrForbidden))
	}

	type TableWithColumns struct {
		*model.TableInfo
		Columns []*model.ColumnInfo `json:"columns"`
	}

	// 获取database同步好的数据记录
	var tables []*TableWithColumns
	var total int64 = 1
	for len(tables) < int(total) {
		var list []*model.TableInfo
		list, total, err = h.tableInfoService.List(c.Context(), &model.TableInfo{
			DataSourceID: dataSource.ID,
		}, len(tables), 100)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
		}

		for _, t := range list {
			twc := new(TableWithColumns)
			twc.TableInfo = t
			var cl []*model.ColumnInfo
			var ct int64 = 1
			for len(cl) < int(ct) {
				var tempList []*model.ColumnInfo
				tempList, ct, err = h.columnInfoService.List(c.Context(), &model.ColumnInfo{
					TableID: t.ID,
				}, len(cl), 100)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
				}
				cl = append(cl, tempList...)
			}
			twc.Columns = cl
			tables = append(tables, twc)
		}
	}

	return c.JSON(service.OK(tables))
}
func (h *DatabaseHandler) ExecuteSqlForDatabase(c *fiber.Ctx) error {
	application, _ := c.Locals("application").(*model.Application)
	if application == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidCredential))
	}

	type Req struct {
		Sql          string `json:"sql"`
		DataSourceID uint64 `json:"datasourceId,string"`
	}
	req := new(Req)
	if err := c.BodyParser(req); err != nil {
		logger.Error("请求参数解析失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidParams))
	}

	if req.Sql == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(service.ErrInvalidParams))
	}

	// 检查datasource是否该应用
	dataSource, err := h.dataSourceService.Get(c.Context(), req.DataSourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
	}
	if dataSource.ApplicationID != application.ID {
		return c.Status(fiber.StatusForbidden).JSON(service.Error(service.ErrForbidden))
	}

	// 得到datasource的连接
	db, err := datasource.GetDB(dataSource)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
	}

	// 进行查询，并把结果放到map[string]interface{}
	var result []map[string]interface{}
	err = db.Raw(req.Sql).Find(&result).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(service.ErrDatabaseError))
	}

	return c.JSON(service.OK(result))
}
