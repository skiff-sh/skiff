package controller

import "github.com/gofiber/fiber/v2"

// Controllers a registry of all controllers within the app.
var Controllers = []Controller{
	new(Hello),
}

type Controller interface {
	Method() string
	Path() string
	Handle(c *fiber.Ctx) error
}
