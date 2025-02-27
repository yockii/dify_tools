package sysapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
)

type KnowledgeBaseHandler struct {
	knowledgeService service.KnowledgeBaseService
	logService       service.LogService
}

func RegisterKnowledgeBaseHandler(knowledgeService service.KnowledgeBaseService, logService service.LogService) {
	handler := &KnowledgeBaseHandler{
		knowledgeService: knowledgeService,
		logService:       logService,
	}
	Handlers = append(Handlers, handler)
}

func (h *KnowledgeBaseHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	knowledgeBaseRouter := router.Group("/knowledge_base")
	knowledgeBaseRouter.Use(authMiddleware)
	{
		knowledgeBaseRouter.Post("/new", h.CreateKnowledgeBase)
		knowledgeBaseRouter.Get("/list", h.GetKnowledgeBaseList)
		// knowledgeBaseRouter.Post("/delete", h.DeleteKnowledgeBase)
	}
}

func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c *fiber.Ctx) error {
	var record model.KnowledgeBase
	if err := c.BodyParser(&record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if record.ApplicationID == 0 || record.KnowledgeBaseName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.knowledgeService.Create(c.Context(), &record); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(record))
}

func (h *KnowledgeBaseHandler) DeleteKnowledgeBase(c *fiber.Ctx) error {
	var record model.KnowledgeBase
	if err := c.BodyParser(&record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if record.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if err := h.knowledgeService.Delete(c.Context(), record.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}
	return c.JSON(service.OK(nil))
}

func (h *KnowledgeBaseHandler) GetKnowledgeBaseList(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var condition model.KnowledgeBase
	if err := c.QueryParser(&condition); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if condition.ApplicationID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	list, total, err := h.knowledgeService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDatabaseError))
	}

	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))

}
