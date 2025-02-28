package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/service"
)

func NewAppMiddleware(
	applicationService service.ApplicationService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 获取api_key, Authorization Bearer
		authorization := c.Get("Authorization")
		apiKey := strings.TrimPrefix(authorization, "Bearer ")

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
		}

		// 检查api_key是否有效
		application, err := applicationService.GetByApiKey(c.Context(), apiKey)
		if err != nil {
			return c.Status(constant.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
		}
		if application == nil || application.Status != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(service.Error(constant.ErrUnauthorized))
		}

		// TODO 后续对限流和源做认证

		// 将应用信息存入上下文
		c.Locals("application", application)

		return c.Next()
	}
}
