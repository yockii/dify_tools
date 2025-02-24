package sysapi

import "github.com/gofiber/fiber/v2"

var Handlers []Handler

type Handler interface {
	RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler)
}
