package sysapi

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type DictHandler struct {
	dictService service.DictService
	logService  service.LogService
}

func RegisterDictHandler(
	dictService service.DictService,
	logService service.LogService,
) {
	handler := &DictHandler{
		dictService: dictService,
		logService:  logService,
	}
	Handlers = append(Handlers, handler)
}

func (h *DictHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	r := router.Group("/dict", authMiddleware)
	{
		r.Post("/new", h.Create)
		r.Post("/update", h.Update)
		r.Post("/delete", h.Delete)
		r.Get("/get", h.Get)
		r.Get("/list", h.List)
		r.Get("/list_by_parent_code", h.ListByParentCode)
	}
}

func (h *DictHandler) Create(c *fiber.Ctx) error {
	record := new(model.Dict)
	if err := c.BodyParser(record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if err := h.dictService.Create(c.Context(), record); err != nil {
		logger.Error("创建字典失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionCreateDict, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(record))
}

func (h *DictHandler) Update(c *fiber.Ctx) error {
	record := new(model.Dict)
	if err := c.BodyParser(record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if err := h.dictService.Update(c.Context(), record); err != nil {
		logger.Error("更新字典失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionUpdateDict, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(record))
}

func (h *DictHandler) Delete(c *fiber.Ctx) error {
	record := new(model.Dict)
	if err := c.BodyParser(record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if err := h.dictService.Delete(c.Context(), record.ID); err != nil {
		logger.Error("删除字典失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	// 记录操作日志
	user := c.Locals("user").(*model.User)
	go h.logService.CreateOperationLog(c.Context(), user.ID, constant.LogActionDeleteDict, c.IP(), c.Get("User-Agent"))

	return c.JSON(service.OK(nil))
}

func (h *DictHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	record, err := h.dictService.Get(c.Context(), id)
	if err != nil {
		logger.Error("获取字典失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	return c.JSON(service.OK(record))
}

func (h *DictHandler) List(c *fiber.Ctx) error {
	condition := new(model.Dict)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	list, total, err := h.dictService.List(c.Context(), condition, offset, limit)
	if err != nil {
		logger.Error("获取字典列表失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))
}

func (h *DictHandler) ListByParentCode(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}
	condition := new(model.Dict)
	if err := c.QueryParser(condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	parentCode := c.Query("parent_code")
	if parentCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	parentDict, err := h.dictService.GetByCode(c.Context(), parentCode)
	if err != nil {
		logger.Error("获取父字典失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	if parentDict == nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrDictNotFound))
	}

	condition.ParentID = parentDict.ID
	list, total, err := h.dictService.List(c.Context(), condition, offset, limit)
	if err != nil {
		logger.Error("获取字典列表失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}
	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))
}
