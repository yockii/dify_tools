package appapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type DocumentHandler struct {
	knowledgeBaseService service.KnowledgeBaseService
	documentService      service.DocumentService
}

func RegisterDocumentHandler(
	knowledgeBaseService service.KnowledgeBaseService,
	documentService service.DocumentService,
) {
	handler := &DocumentHandler{
		knowledgeBaseService: knowledgeBaseService,
		documentService:      documentService,
	}
	Handlers = append(Handlers, handler)
}

func (h *DocumentHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/document/add", h.AddDocument)
	router.Get("/document/status", h.DocumentStatus)
	router.Post("/document/delete", h.DeleteDocument)
}

func (h *DocumentHandler) AddDocument(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}

	if form, err := c.MultipartForm(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	} else {
		customIDFV := form.Value["custom_id"]
		customID := ""
		if len(customIDFV) > 0 {
			customID = customIDFV[0]
		}
		files := form.File["files"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		file := files[0]
		document := &model.Document{
			ApplicationID:   application.ID,
			KnowledgeBaseID: 0,
			CustomID:        customID,
			FileName:        file.Filename,
			FileSize:        file.Size,
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

func (h *DocumentHandler) DocumentStatus(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}
	var document model.Document
	if err := c.BodyParser(&document); err != nil {
		logger.Error("解析字典参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if document.ID == 0 && document.OuterID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	document.ApplicationID = application.ID
	doc, err := h.documentService.GetDocument(c.Context(), &document)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(service.Error(constant.ErrRecordNotFound))
	}

	return c.JSON(service.OK(doc))
}

func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}

	var document model.Document
	if err := c.BodyParser(&document); err != nil {
		logger.Error("解析字典参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if document.ID == 0 && document.OuterID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	document.ApplicationID = application.ID
	doc, err := h.documentService.GetDocument(c.Context(), &document)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(service.Error(constant.ErrRecordNotFound))
	}

	err = h.documentService.Delete(c.Context(), doc.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(true))
}
