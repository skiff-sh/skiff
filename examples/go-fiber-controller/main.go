package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/skiff-sh/skiff/examples/go-fiber-controller/controller"
)

func main() {
	app := fiber.New()

	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.SendString("healthy")
	})

	for _, v := range controller.Controllers {
		app.Add(v.Method(), v.Path(), v.Handle)
	}

	log.Fatal(app.Listen(":8080"))
}
