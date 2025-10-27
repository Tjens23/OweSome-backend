package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
)

func main() {
    app := fiber.New()
	database.Connect()
    app.Get("/", func(c fiber.Ctx) error {
        return c.JSON((fiber.Map{
			"message": "Hello, World!",
			"status":  fiber.StatusOK,
		}))
    })
    log.Fatal(app.Listen(":3001"))
}