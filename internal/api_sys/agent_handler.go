package sysapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type AgentHandler struct {
	agentService service.AgentService
}

func RegisterAgentHandler(agentService service.AgentService) {
	handler := &AgentHandler{
		agentService: agentService,
	}
	Handlers = append(Handlers, handler)
}

func (h *AgentHandler) RegisterRoutesV1_1(router fiber.Router, authMiddleware fiber.Handler) {
	h.RegisterRoutesV1(router, authMiddleware)
}

func (h *AgentHandler) RegisterRoutesV1(router fiber.Router, authMiddleware fiber.Handler) {
	agentRouter := router.Group("/agent")
	agentRouter.Use(authMiddleware)
	{
		agentRouter.Post("/new", h.CreateAgent)
		agentRouter.Get("/list", h.GetAgentList)
		agentRouter.Post("/delete", h.DeleteAgent)
		agentRouter.Post("/update", h.UpdateAgent)
	}
}

func (h *AgentHandler) CreateAgent(c *fiber.Ctx) error {
	var record model.Agent
	if err := c.BodyParser(&record); err != nil {
		logger.Error("解析参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if record.Code == "" || record.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.agentService.Create(c.Context(), &record); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(record))
}

func (h *AgentHandler) UpdateAgent(c *fiber.Ctx) error {
	var record model.Agent
	if err := c.BodyParser(&record); err != nil {
		logger.Error("解析参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if record.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if err := h.agentService.Update(c.Context(), &record); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.OK(record))
}

func (h *AgentHandler) DeleteAgent(c *fiber.Ctx) error {
	var record model.Agent
	if err := c.BodyParser(&record); err != nil {
		logger.Error("解析参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if record.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if err := h.agentService.Delete(c.Context(), record.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}
	return c.JSON(service.OK(true))
}

func (h *AgentHandler) GetAgentList(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", service.DefaultPageSize)
	if limit > service.MaxPageSize {
		limit = service.MaxPageSize
	}

	var condition model.Agent
	if err := c.QueryParser(&condition); err != nil {
		logger.Error("解析参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	list, total, err := h.agentService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(err))
	}

	return c.JSON(service.NewListResponse(list, total, offset, limit))
}
