package controller

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

var _ Controller = (*Hello)(nil)

type Hello struct {
}

func (h *Hello) Method() string {
	return http.MethodGet
}

func (h *Hello) Path() string {
	return "/v1/hello"
}

func (h *Hello) Handle(c *fiber.Ctx) error {
	return c.SendString("hi!")
}
