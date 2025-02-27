package sysapi

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/util"
)

type AppHandler struct {
	appService        service.ApplicationService
	dataSourceService service.DataSourceService
	tableInfoService  service.TableInfoService
	columnInfoService service.ColumnInfoService

	knowledgeService service.KnowledgeBaseService

	logService service.LogService
}

func RegisterAppHandler(
	applicationService service.ApplicationService,
	dataSourceService service.DataSourceService,
	tableInfoService service.TableInfoService,
	columnInfoService service.ColumnInfoService,
	knowledgeService service.KnowledgeBaseService,

	logService service.LogService,
) {
	handler := &AppHandler{
		appService:        applicationService,
		dataSourceService: dataSourceService,
		tableInfoService:  tableInfoService,
		columnInfoService: columnInfoService,
		knowledgeService:  knowledgeService,

		logService: logService,
	}
	Handlers = append(Handlers, handler)
}

func (h *AppHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	apps := router.Group("/applications", authMiddleware)
	{
		apps.Post("/new", h.CreateApp)
		apps.Post("/update", h.UpdateApp)
		apps.Post("/delete", h.DeleteApp)
		apps.Get("/list", h.ListApps)
		apps.Get("/info", h.GetApp)
	}

	dataSources := router.Group("/data_sources", authMiddleware)
	{
		dataSources.Post("/new", h.CreateDataSource)
		dataSources.Post("/update", h.UpdateDataSource)
		dataSources.Post("/delete", h.DeleteDataSource)
		dataSources.Get("/list", h.ListDataSources)
		dataSources.Get("/info", h.GetDataSource)
		dataSources.Get("/sync", h.SyncDataSource)
		dataSources.Get("/tables", h.GetDataSourceTables)
		dataSources.Post("/update_table", h.UpdateDataSourceTable)
		dataSources.Get("/columns", h.GetDataSourceColumns)
		dataSources.Post("/update_column", h.UpdateDataSourceColumn)

	}
}

///////////////////////////////////////////////////////////////////
//////////               Application                     //////////
//region///////////////////////////////////////////////////////////

// CreateApp 创建应用
func (h *AppHandler) CreateApp(c *fiber.Ctx) error {
	var app model.Application
	if err := c.BodyParser(&app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	app.Status = 1
	app.APIKey = "ak-" + util.NewShortID()

	if err := h.appService.Create(c.Context(), &app); err != nil {
		logger.Error("创建应用失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	go h.knowledgeService.Create(c.Context(), &model.KnowledgeBase{
		ApplicationID: app.ID,
	})

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionCreateApplication, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(app))
}

// UpdateApp 更新应用
func (h *AppHandler) UpdateApp(c *fiber.Ctx) error {
	var app model.Application
	if err := c.BodyParser(&app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	app.APIKey = "" // 禁止修改APIKey

	if err := h.appService.Update(c.Context(), &app); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateApplication, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(app))
}

// DeleteApp 删除应用
func (h *AppHandler) DeleteApp(c *fiber.Ctx) error {
	var app model.Application
	if err := c.BodyParser(&app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if err := h.appService.Delete(c.Context(), app.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 删除知识库
	go h.knowledgeService.DeleteByApplicationID(c.Context(), app.ID)

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionDeleteApplication, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// ListApps 获取应用列表
func (h *AppHandler) ListApps(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var condition model.Application
	if err := c.QueryParser(&condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	apps, total, err := h.appService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	return c.JSON(service.OK(service.NewListResponse(apps, total, offset, limit)))
}

// GetApp 获取应用详情
func (h *AppHandler) GetApp(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	app, err := h.appService.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(app))
}

//endregion

///////////////////////////////////////////////////////////////////
//////////               DataSource                      //////////
//region///////////////////////////////////////////////////////////

// CreateDataSource 创建数据源
func (h *AppHandler) CreateDataSource(c *fiber.Ctx) error {
	var dataSource model.DataSource
	if err := c.BodyParser(&dataSource); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if dataSource.ApplicationID == 0 || dataSource.Name == "" || dataSource.Host == "" || dataSource.Port == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.dataSourceService.Create(c.Context(), &dataSource); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionCreateDataSource, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(dataSource))
}

// UpdateDataSource 更新数据源
func (h *AppHandler) UpdateDataSource(c *fiber.Ctx) error {
	var dataSource model.DataSource
	if err := c.BodyParser(&dataSource); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if err := h.dataSourceService.Update(c.Context(), &dataSource); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateDataSource, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(dataSource))
}

// DeleteDataSource 删除数据源
func (h *AppHandler) DeleteDataSource(c *fiber.Ctx) error {
	var dataSource model.DataSource
	if err := c.BodyParser(&dataSource); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if err := h.dataSourceService.Delete(c.Context(), dataSource.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionDeleteDataSource, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// ListDataSources 获取数据源列表
func (h *AppHandler) ListDataSources(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var condition model.DataSource
	if err := c.QueryParser(&condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	dataSources, total, err := h.dataSourceService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(dataSources, total, offset, limit)))
}

// GetDataSource 获取数据源详情
func (h *AppHandler) GetDataSource(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	dataSource, err := h.dataSourceService.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(dataSource))
}

// SyncDataSource 同步数据源
func (h *AppHandler) SyncDataSource(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.dataSourceService.Sync(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionSyncDataSource, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

// GetDataSourceTables 获取数据源表列表
func (h *AppHandler) GetDataSourceTables(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	condition := new(model.TableInfo)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}
	if condition.DataSourceID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	tables, total, err := h.tableInfoService.List(c.Context(), condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(tables, total, offset, limit)))
}

// UpdateDataSourceTable 更新数据源表
func (h *AppHandler) UpdateDataSourceTable(c *fiber.Ctx) error {
	var table model.TableInfo
	if err := c.BodyParser(&table); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if err := h.tableInfoService.Update(c.Context(), &table); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateTableInfo, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(table))
}

// GetDataSourceColumns 获取数据源表列列表
func (h *AppHandler) GetDataSourceColumns(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	condition := new(model.ColumnInfo)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}
	if condition.TableID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	columns, total, err := h.columnInfoService.List(c.Context(), condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(columns, total, offset, limit)))
}

// UpdateDataSourceColumn 更新数据源表列
func (h *AppHandler) UpdateDataSourceColumn(c *fiber.Ctx) error {
	var column model.ColumnInfo
	if err := c.BodyParser(&column); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(err))
	}

	if err := h.columnInfoService.Update(c.Context(), &column); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateColumnInfo, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(column))
}

//endregion
