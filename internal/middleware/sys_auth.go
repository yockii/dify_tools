package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(
	authService service.AuthService,
	sessionService service.SessionService,
	skipPaths []string,
) fiber.Handler {
	skipPathMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipPathMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		// 检查是否跳过认证
		if skipPathMap[c.Path()] {
			return c.Next()
		}

		// 获取token
		authorization := c.Get("Authorization")
		token := strings.TrimPrefix(authorization, "Bearer ")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(service.NewResponse(nil, service.ErrUnauthorized))
		}

		// 检查token是否在黑名单中
		if sessionService != nil && sessionService.IsTokenBlocked(c.Context(), token) {
			logger.Warn("blocked token used", logger.F("token", token))
			return c.Status(fiber.StatusUnauthorized).JSON(service.NewResponse(nil, service.ErrUnauthorized))
		}

		// 验证token
		user, err := authService.Verify(c.Context(), token)
		if err != nil {
			return c.Status(service.GetErrorCode(err)).JSON(service.NewResponse(nil, err))
		}

		// 将用户信息存入上下文
		c.Locals("user", user)

		// 刷新会话
		if sessionService != nil {
			if err := sessionService.RefreshSession(c.Context(), token); err != nil {
				logger.Warn("refresh session failed", logger.F("error", err))
			}
		}

		return c.Next()
	}
}

// // RequirePermissions 权限检查中间件
// func RequirePermissions(permissions ...string) fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		// 获取用户信息
// 		user, ok := c.Locals("user").(*model.User)
// 		if !ok || user == nil {
// 			return c.Status(fiber.StatusUnauthorized).JSON(service.Error(service.ErrUnauthorized))
// 		}
//
// 		// 超级管理员拥有所有权限
// 		if user.Role != nil && user.Role.Code == "ADMIN" {
// 			return c.Next()
// 		}
//
// 		// 检查用户权限
// 		if !hasPermissions(user.Role.Permissions, permissions) {
// 			// 获取用户权限代码列表用于日志
// 			var userPermCodes []string
// 			if user.Role != nil && user.Role.Permissions != nil {
// 				for _, p := range user.Role.Permissions {
// 					userPermCodes = append(userPermCodes, p.Code)
// 				}
// 			}
//
// 			logger.Warn("permission denied",
// 				logger.F("user", user.Username),
// 				logger.F("required", permissions),
// 				logger.F("has", userPermCodes),
// 			)
// 			return c.Status(fiber.StatusForbidden).JSON(service.Error(service.ErrPermissionDenied))
// 		}
//
// 		return c.Next()
// 	}
// }

// hasPermissions 检查用户是否拥有所需权限
func hasPermissions(userPerms []*model.Permission, requiredPerms []string) bool {
	if len(requiredPerms) == 0 {
		return true
	}
	if len(userPerms) == 0 {
		return false
	}

	// 转换为map便于查找
	permMap := make(map[string]bool)
	for _, p := range userPerms {
		permMap[p.Code] = true

		// 支持通配符权限，例如 app:* 可以匹配所有 app: 开头的权限
		if strings.HasSuffix(p.Code, ":*") {
			permMap[strings.TrimSuffix(p.Code, "*")] = true
		}
	}

	// 检查每个所需权限
	for _, required := range requiredPerms {
		// 完整匹配
		if permMap[required] {
			return true
		}

		// 通配符匹配
		parts := strings.Split(required, ":")
		prefix := ""
		for i := 0; i < len(parts)-1; i++ {
			prefix += parts[i] + ":"
			// 检查是否有对应的通配符权限
			if permMap[prefix] {
				return true
			}
		}
	}

	return false
}

// Recovery 错误恢复中间件
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					logger.F("error", r),
					logger.F("path", c.Path()),
					logger.F("method", c.Method()),
				)

				err, ok := r.(error)
				if !ok {
					err = fiber.ErrInternalServerError
				}

				c.Status(fiber.StatusInternalServerError).JSON(service.NewResponse(nil, err))
			}
		}()

		return c.Next()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		err := c.Next()

		// 记录请求日志
		logger.Info("request completed",
			logger.F("method", c.Method()),
			logger.F("path", c.Path()),
			logger.F("status", c.Response().StatusCode()),
			logger.F("duration", time.Since(start)),
			logger.F("ip", c.IP()),
			logger.F("user-agent", c.Get("User-Agent")),
		)

		return err
	}
}
