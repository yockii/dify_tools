package appapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
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
	router.Post("/add_document", h.AddDocument)
	router.Get("/document_status", h.DocumentStatus)
	router.Post("/delete_document", h.DeleteDocument)
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
	return nil
}

func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	return nil
}
