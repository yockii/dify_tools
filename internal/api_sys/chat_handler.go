package sysapi

import (
	"bufio"
	"bytes"
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/dify"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/pptgen"
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

func (h *ChatHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	chatRouter := router.Group("/chat")
	chatRouter.Use(authMiddleware)
	{
		chatRouter.Post("/send", h.SendMessage)
		chatRouter.Get("/list", h.GetSessionList)
		chatRouter.Get("/history", h.GetSessionHistory)
		chatRouter.Post("/stop", h.StopChatFlow)
		chatRouter.Post("/generate_ppt", h.GeneratePPT)
	}
}

type ChatMessageRequest struct {
	Query          string `json:"query"`
	ConversationID string `json:"conversation_id"`
	AppSecret      string `json:"app_secret"`
	AgentID        uint64 `json:"agent_id,string,omitzero"`
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	if user == nil {
		logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
	}

	var req ChatMessageRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	var appID uint64

	customID := ""
	appSecret := ""
	if req.AppSecret == "" {
		customID = strconv.FormatUint(user.ID, 10)
	} else {
		appSecret = req.AppSecret
		// 获取对应的app
		app, err := h.applicationService.GetByApiKey(c.Context(), req.AppSecret)
		if err != nil {
			logger.Error("获取应用失败", logger.F("err", err))
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
		}
		if app == nil {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}
		appID = app.ID
	}

	chatMessageRequest := &dify.ChatMessageRequest{
		User:         strconv.FormatUint(user.ID, 10),
		ResponseMode: "streaming",
		Inputs: map[string]interface{}{
			"custom_id":  customID,
			"app_secret": appSecret,
		},
		Query:            req.Query,
		ConversationID:   req.ConversationID,
		AutoGenerateName: true,
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		logger.Error("获取dify chat client失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), appID, req.AgentID)
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
			// 检查是否是结束信号
			if bytes.Contains(data, []byte(`"event":"end"`)) {
				// 这是结束信号，发送一个最终的SSE消息给客户端表示流结束
				if _, err := w.Write([]byte("data: {\"event\":\"done\"}\n\n")); err != nil {
					logger.Error("发送结束信号失败", logger.F("err", err))
					return err
				}
				if err := w.Flush(); err != nil {
					logger.Error("发送结束信号失败", logger.F("err", err))
					return err
				}
				return nil
			}

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

type StopStreamingChatRequest struct {
	TaskID string `json:"task_id"`
}

func (h *ChatHandler) StopChatFlow(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	if user == nil {
		logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
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

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), 0, 0)
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	err = chatClient.StopStreamingChat(req.TaskID, strconv.FormatUint(user.ID, 10), appAgent.Agent.ApiSecret)
	if err != nil {
		logger.Error("停止会话失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(true))
}

type SessionListRequest struct {
	AppSecret string `json:"app_secret"`
	AgentID   uint64 `json:"agent_id,string,omitzero"`
}

func (h *ChatHandler) GetSessionList(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	if user == nil {
		logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
	}

	var req SessionListRequest
	if err := c.QueryParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	var appID uint64
	if req.AppSecret != "" {
		app, err := h.applicationService.GetByApiKey(c.Context(), req.AppSecret)
		if err != nil {
			logger.Error("获取应用失败", logger.F("err", err))
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
		}
		if app == nil {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}
		appID = app.ID
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), appID, req.AgentID)
	if err != nil {
		logger.Error("获取应用代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	apiSecret := ""
	if appAgent != nil {
		apiSecret = appAgent.Agent.ApiSecret
	}

	list, err := chatClient.GetConversations(strconv.FormatUint(user.ID, 10), apiSecret)
	if err != nil {
		logger.Error("获取会话列表失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(list))
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

func (h *ChatHandler) GetSessionHistory(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	if user == nil {
		logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), 0, 0)
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
	firstID := c.Query("first_id")

	limit := c.QueryInt("limit", 20)

	historyData, err := chatClient.GetConversationHistory(conversationID, strconv.FormatUint(user.ID, 10), firstID, limit, apiSecret)
	if err != nil {
		logger.Error("获取会话历史失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(historyData))
}

type GeneratePPTRequest struct {
	TemplateType string `json:"template_type"`
	ThemeColor   string `json:"theme_color"`
	FontFamily   string `json:"font_family"`
	Content      string `json:"content"`
}

func (h *ChatHandler) GeneratePPT(c *fiber.Ctx) error {
	// user := c.Locals("user").(*model.User)
	// if user == nil {
	// 	logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
	// 	return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
	// }

	var req GeneratePPTRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	// 检查和映射模板类型
	var templateEnum pptgen.TemplateType
	switch req.TemplateType {
	case "business":
		templateEnum = pptgen.TemplateBusiness
	case "academic":
		templateEnum = pptgen.TemplateAcademic
	case "minimalist":
		templateEnum = pptgen.TemplateMinimalist
	default:
		logger.Error("无效的模板类型", logger.F("template_type", req.TemplateType))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	pptGenerator := pptgen.NewPPTGenerator()
	config := pptgen.TemplateConfig{
		Type:       templateEnum,
		ThemeColor: req.ThemeColor,
		FontFamily: req.FontFamily,
	}

	buf, err := pptGenerator.GeneratePPTX(config, req.Content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	// 下载文件流
	c.Set("Content-Disposition", "attachment; filename=generated.pptx")
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Set("Content-Length", strconv.Itoa(len(buf)))
	return c.Status(fiber.StatusOK).Send(buf)
}
