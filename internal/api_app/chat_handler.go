﻿package appapi

import (
	"bufio"
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/dify"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type ChatHandler struct {
	dictService        service.DictService
	applicationService service.ApplicationService
	usageService       service.UsageService
}

func RegisterChatHandler(
	dictService service.DictService,
	applicationService service.ApplicationService,
	usageService service.UsageService,
) {
	handler := &ChatHandler{
		dictService:        dictService,
		applicationService: applicationService,
		usageService:       usageService,
	}
	Handlers = append(Handlers, handler)
}

func (h *ChatHandler) RegisterRoutes(router fiber.Router) {
	chatRouter := router.Group("/chat")
	{
		chatRouter.Post("/send", h.SendMessage)
		chatRouter.Get("/list", h.GetSessionList)
		chatRouter.Get("/history", h.GetSessionHistory)
		chatRouter.Post("/stop", h.StopChatFlow)
		chatRouter.Get("/usage", h.GetUsage)
	}
}

type ChatMessageRequest struct {
	AgentID        uint64 `json:"agent_id,string"`
	CustomID       string `json:"custom_id"`
	Query          string `json:"query"`
	ConversationID string `json:"conversation_id"`
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}
	var req ChatMessageRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	chatMessageRequest := &dify.ChatMessageRequest{
		User:         req.CustomID,
		Query:        req.Query,
		ResponseMode: "streaming",
		Inputs: map[string]interface{}{
			"custom_id":  req.CustomID,
			"app_secret": application.APIKey,
		},
		ConversationID:   req.ConversationID,
		AutoGenerateName: true,
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		logger.Error("获取dify chat client失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, req.AgentID)
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	apiSecret := ""
	if appAgent != nil {
		apiSecret = appAgent.Agent.ApiSecret
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Status(fiber.StatusOK).Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		_, err = chatClient.SendChatMessage(chatMessageRequest, apiSecret, func(data []byte) error {
			go h.usageService.CreateByEndMessage(0, string(data))

			if _, err := w.Write(append([]byte("data: "), data...)); err != nil {
				logger.Error("发送消息失败", logger.F("err", err))
				return err
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				logger.Error("发送消息失败", logger.F("err", err))
				return err
			}
			if err := w.Flush(); err != nil {
				logger.Error("发送消息失败", logger.F("err", err))
				return err
			}
			return nil
		})
		if err != nil {
			logger.Error("发送消息失败", logger.F("err", err))
			return
		}
	}))
	return nil
}

func (h *ChatHandler) GetDifyChatClient(ctx context.Context) (*dify.ChatClient, error) {
	chatClient := dify.GetDefaultChatClient()
	if chatClient == nil {
		difyBaseUrlDict, err := h.dictService.GetByCode(ctx, constant.DictCodeDifyBaseUrl)
		if err != nil {
			logger.Error("获取字典值失败", logger.F("err", err))
			return nil, err
		}
		if difyBaseUrlDict == nil || difyBaseUrlDict.Value == "" {
			logger.Warn("未配置dify接口地址", logger.F("dict_id", difyBaseUrlDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		difyBaseUrl := difyBaseUrlDict.Value
		difyCommonFlowTokenDict, err := h.dictService.GetByCode(ctx, constant.DictCodeDifyToken)
		if err != nil {
			logger.Error("获取字典值失败", logger.F("err", err))
			return nil, err
		}
		difyToken := ""
		if difyCommonFlowTokenDict != nil && difyCommonFlowTokenDict.Value != "" {
			difyToken = difyCommonFlowTokenDict.Value
		}
		chatClient = dify.InitDefaultChatClient(difyBaseUrl, difyToken)
	}
	return chatClient, nil
}

func (h *ChatHandler) GetSessionList(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}
	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	agentID := c.QueryInt("agent_id", 0)
	customID := c.Query("custom_id")

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, uint64(agentID))
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	apiSecret := ""
	if appAgent != nil {
		apiSecret = appAgent.Agent.ApiSecret
	}

	list, err := chatClient.GetConversations(customID, apiSecret)
	if err != nil {
		logger.Error("获取会话列表失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(list))
}

func (h *ChatHandler) GetSessionHistory(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}

	agentID := c.QueryInt("agent_id", 0)

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, uint64(agentID))
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	apiSecret := ""
	if appAgent != nil {
		apiSecret = appAgent.Agent.ApiSecret
	}

	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	customID := c.Query("custom_id")
	firstID := c.Query("first_id")

	limit := c.QueryInt("limit", 20)

	historyData, err := chatClient.GetConversationHistory(conversationID, customID, firstID, limit, apiSecret)
	if err != nil {
		logger.Error("获取会话历史失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(historyData))
}

type StopStreamingChatRequest struct {
	AgentID  uint64 `json:"agent_id,string"`
	CustomID string `json:"custom_id"`
	TaskID   string `json:"task_id"`
}

func (h *ChatHandler) StopChatFlow(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}

	var req StopStreamingChatRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if req.TaskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, req.AgentID)
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	err = chatClient.StopStreamingChat(req.TaskID, req.CustomID, appAgent.Agent.ApiSecret)
	if err != nil {
		logger.Error("停止会话失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(true))
}

func (h *ChatHandler) GetUsage(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}
	if application.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	var condition model.Usage
	if err := c.QueryParser(&condition); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	condition.ApplicationID = application.ID

	list, total, err := h.usageService.List(c.Context(), &condition, offset, limit)
	if err != nil {
		logger.Error("获取使用记录失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(service.NewListResponse(list, total, offset, limit)))
}
