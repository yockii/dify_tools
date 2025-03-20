package difyapi

import "github.com/gofiber/fiber/v2"

var Handlers []Handler

type Handler interface {
	RegisterRoutesV1(router fiber.Router)
	RegisterRoutesV1_1(router fiber.Router)
}
