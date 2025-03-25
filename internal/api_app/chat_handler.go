package appapi

import (
	"bufio"
	"bytes"
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tidwall/gjson"
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

func (h *ChatHandler) RegisterRoutes(router fiber.Router) {
	chatRouter := router.Group("/chat")
	{
		chatRouter.Post("/send", h.SendMessage)
		chatRouter.Get("/list", h.GetSessionList)
		chatRouter.Get("/history", h.GetSessionHistory)
		chatRouter.Post("/stop", h.StopChatFlow)
		chatRouter.Get("/usage", h.GetUsage)

		chatRouter.Post("/generate_ppt", h.GeneratePPT)

		chatRouter.Post("/del_conversation", h.DeleteConversation)

		chatRouter.Post("/upload", h.UploadFile)
	}
	router.Get("/files_proxy/*", h.FileProxy)
}

type ChatMessageRequest struct {
	AgentID        uint64          `json:"agent_id,string"`
	CustomID       string          `json:"custom_id"`
	Query          string          `json:"query"`
	ConversationID string          `json:"conversation_id"`
	Files          []dify.ChatFile `json:"files"`
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
		Files:            req.Files,
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		logger.Error("获取dify chat client失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, req.AgentID)
	if err != nil {
		logger.Error("获取应用智能体失败", logger.F("err", err))
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

			go h.usageService.CreateByEndMessage(appAgent.ApplicationID, appAgent.AgentID, string(data))

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
		logger.Error("获取应用智能体失败", logger.F("err", err))
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
		logger.Error("获取应用智能体失败", logger.F("err", err))
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
		logger.Error("获取应用智能体失败", logger.F("err", err))
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

type GeneratePPTRequest struct {
	TemplateType string `json:"template_type"`
	ThemeColor   string `json:"theme_color"`
	FontFamily   string `json:"font_family"`
	Content      string `json:"content"`
}

func (h *ChatHandler) GeneratePPT(c *fiber.Ctx) error {
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

type SessionListRequest struct {
	CustomID string `json:"custom_id"`
	AgentID  uint64 `json:"agent_id,string,omitzero"`
}

func (h *ChatHandler) UploadFile(c *fiber.Ctx) error {
	if form, err := c.MultipartForm(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	} else {
		application, ok := c.Locals("application").(*model.Application)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
		}
		if application.ID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		var req SessionListRequest
		if err := c.QueryParser(&req); err != nil {
			logger.Error("解析请求参数失败", logger.F("err", err))
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, req.AgentID)
		if err != nil {
			logger.Error("获取应用智能体失败", logger.F("err", err))
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
		}
		apiSecret := ""
		if appAgent != nil {
			apiSecret = appAgent.Agent.ApiSecret
		}

		files := form.File["files"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
		}

		file := files[0]

		chatClient, err := h.GetDifyChatClient(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
		}

		response, err := chatClient.UploadFile(file, apiSecret, req.CustomID)
		if err != nil {
			logger.Error("上传文件失败", logger.F("err", err))
			return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
		}

		j := gjson.Parse(response)
		// 转为map[string]string
		m := make(map[string]string)
		j.ForEach(func(key, value gjson.Result) bool {
			m[key.String()] = value.String()
			return true
		})

		return c.JSON(service.OK(m))
	}
}

func (h *ChatHandler) FileProxy(c *fiber.Ctx) error {
	targetURI := c.Params("*") + "?" + string(c.Request().URI().QueryString())
	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	err = chatClient.ProxyFile(targetURI, c)
	if err != nil {
		logger.Error("文件代理失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return nil
}

type DeleteConversationRequest struct {
	ConversationID string
	AgentID        uint64
	CustomID       string
}

func (r *DeleteConversationRequest) UnmarshalJSON(data []byte) error {
	j := gjson.Parse(string(data))
	r.ConversationID = j.Get("conversation_id").String()
	r.AgentID = j.Get("agent_id").Uint()
	r.CustomID = j.Get("custom_id").String()
	return nil
}

func (h *ChatHandler) DeleteConversation(c *fiber.Ctx) error {
	application, ok := c.Locals("application").(*model.Application)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
	}
	if application.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	var req DeleteConversationRequest

	if err := c.BodyParser(&req); err != nil {
		logger.Error("解析请求参数失败", logger.F("err", err))
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	if req.ConversationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(service.Error(constant.ErrInvalidParams))
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	appAgent, err := h.applicationService.GetApplicationAgent(c.Context(), application.ID, req.AgentID)
	if err != nil {
		logger.Error("获取应用智能体失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}
	apiSecret := ""
	if appAgent != nil {
		apiSecret = appAgent.Agent.ApiSecret
	}

	_, err = chatClient.DeleteConversation(req.ConversationID, req.CustomID, apiSecret)
	if err != nil {
		logger.Error("删除会话失败", logger.F("err", err))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrInternalError))
	}

	return c.JSON(service.OK(true))
}
