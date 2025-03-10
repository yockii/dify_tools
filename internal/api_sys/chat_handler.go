package sysapi

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/dify"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type ChatHandler struct {
	dictService service.DictService
}

func RegisterChatHandler(
	dictService service.DictService,
) {
	handler := &ChatHandler{
		dictService: dictService,
	}
	Handlers = append(Handlers, handler)
}

func (h *ChatHandler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	chatRouter := router.Group("/chat")
	chatRouter.Use(authMiddleware)
	{
		chatRouter.Post("/send", h.SendMessage)
		chatRouter.Get("/list", h.GetSessionList)
	}
}

type ChatMessageRequest struct {
	Query          string `json:"query"`
	ConversationID string `json:"conversation_id"`
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {

	return nil
}

func (h *ChatHandler) GetSessionList(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.User)
	if user == nil {
		logger.Error("获取用户信息失败", logger.F("err", "user is nil"))
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrUnauthorized))
	}

	chatClient, err := h.GetDifyChatClient(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(service.Error(constant.ErrDictNotConfigured))
	}

	list, err := chatClient.GetConversations(strconv.FormatUint(user.ID, 10))
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
		if difyCommonFlowTokenDict == nil || difyCommonFlowTokenDict.Value == "" {
			logger.Warn("未配置dify通用流程接口API密钥", logger.F("dict_id", difyCommonFlowTokenDict.ID))
			return nil, constant.ErrDictNotConfigured
		}
		difyToken := difyCommonFlowTokenDict.Value
		chatClient = dify.InitDefaultChatClient(difyBaseUrl, difyToken)
	}
	return chatClient, nil
}
