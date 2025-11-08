package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	routes "github.com/tjens23/tabsplit-backend/src/Routes"
)

func main() {
	app := fiber.New()
	database.Connect()
	routes.SetupUserRoutes(app)
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON((fiber.Map{
			"message": "Hello, World!",
			"status":  fiber.StatusOK,
		}))
	})
	log.Fatal(app.Listen(":3001"))
}
