package sysapi

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type KnowledgeBaseHandler struct {
	knowledgeService service.KnowledgeBaseService
	documentService  service.DocumentService
	logService       service.LogService
}

func RegisterKnowledgeBaseHandler(knowledgeService service.KnowledgeBaseService, documentService service.DocumentService, logService service.LogService) {
	handler := &KnowledgeBaseHandler{
		knowledgeService: knowledgeService,
		documentService:  documentService,
		logService:       logService,
	}
	Handlers = append(Handlers, handler)
}

func (h *KnowledgeBaseHandler) RegisterRoutesV1_1(router fiber.Router, authMiddleware fiber.Handler) {
	knowledgeBaseRouter := router.Group("/knowledge_base")
	knowledgeBaseRouter.Use(authMiddleware)
	{
		knowledgeBaseRouter.Post("/new", h.CreateKnowledgeBase)
		knowledgeBaseRouter.Get("/list", h.GetKnowledgeBaseList)
		// knowledgeBaseRouter.Post("/delete", h.DeleteKnowledgeBase)
	}
	documentRouter := router.Group("/document")
	documentRouter.Use(authMiddleware)
	{
		documentRouter.Post("/upload", h.CreateDocumentV1_1)
		documentRouter.Get("/list", h.GetDocumentList)
		documentRouter.Post("/delete", h.DeleteDocument)
	}
}

func (h *KnowledgeBaseHandler) RegisterRoutesV1(router fiber.Router, authMiddleware fiber.Handler) {
	knowledgeBaseRouter := router.Group("/knowledge_base")
	knowledgeBaseRouter.Use(authMiddleware)
	{
		knowledgeBaseRouter.Post("/new", h.CreateKnowledgeBase)
		knowledgeBaseRouter.Get("/list", h.GetKnowledgeBaseList)
		// knowledgeBaseRouter.Post("/delete", h.DeleteKnowledgeBase)
	}
	documentRouter := router.Group("/document")
	documentRouter.Use(authMiddleware)
	{
		documentRouter.Post("/upload", h.CreateDocument)
		documentRouter.Get("/list", h.GetDocumentList)
		documentRouter.Post("/delete", h.DeleteDocument)
	}
}

func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c *fiber.Ctx) error {
	var record model.KnowledgeBase
	if err := c.BodyParser(&record); err != nil {
		logger.Error("解析字典参数失败", logger.F("err", err))
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
		logger.Error("解析字典参数失败", logger.F("err", err))
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
		logger.Error("解析字典参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if condition.ApplicationID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	list, total, err := h.knowledgeService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))

}

func (h *KnowledgeBaseHandler) CreateDocumentV1_1(c *fiber.Ctx) error {
	if form, err := c.MultipartForm(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	} else {
		user := c.Locals("user").(*model.User)
		customID := strconv.FormatUint(user.ID, 10)

		files := form.File["files"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		file := files[0]
		document := &model.Document{
			CustomID: customID,
			FileName: file.Filename,
			FileSize: file.Size,
		}
		knowledgeBase, err := h.documentService.AddDocumentV1_1(c.Context(), document, file)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
		}

		return c.JSON(service.OK(fiber.Map{
			"knowledge_base": knowledgeBase,
			"document":       document,
		}))
	}
}

func (h *KnowledgeBaseHandler) CreateDocument(c *fiber.Ctx) error {
	if form, err := c.MultipartForm(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	} else {
		user := c.Locals("user").(*model.User)
		customID := strconv.FormatUint(user.ID, 10)

		files := form.File["files"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		file := files[0]
		document := &model.Document{
			CustomID: customID,
			FileName: file.Filename,
			FileSize: file.Size,
		}
		knowledgeBase, err := h.documentService.AddDocument(c.Context(), document, file)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
		}

		return c.JSON(service.OK(fiber.Map{
			"knowledge_base": knowledgeBase,
			"document":       document,
		}))
	}
}

func (h *KnowledgeBaseHandler) DeleteDocument(c *fiber.Ctx) error {
	var document model.Document
	if err := c.BodyParser(&document); err != nil {
		logger.Error("解析字典参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if document.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.documentService.Delete(c.Context(), document.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}
	return c.JSON(service.OK(nil))
}

func (h *KnowledgeBaseHandler) GetDocumentList(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var condition model.Document
	if err := c.QueryParser(&condition); err != nil {
		logger.Error("解析字典参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	user := c.Locals("user").(*model.User)
	customID := strconv.FormatUint(user.ID, 10)
	condition.CustomID = customID

	list, total, err := h.documentService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))
}
