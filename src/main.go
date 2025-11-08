package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	routes "github.com/tjens23/tabsplit-backend/src/Routes"
	_ "github.com/tjens23/tabsplit-backend/src/docs"
)

// @title OweSome Backend API
// @version 1.0
// @description This is the backend API for the OweSome expense splitting application
// @host localhost:3001
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in cookie
// @name jwt

func main() {
	app := fiber.New()
	database.Connect()
	
	// Add Swagger JSON endpoint
	app.Get("/swagger/doc.json", func(c fiber.Ctx) error {
		jsonData, err := os.ReadFile("src/docs/swagger.json")
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Swagger documentation not found",
			})
		}
		c.Set("Content-Type", "application/json")
		return c.Send(jsonData)
	})

	// Add Swagger UI HTML page
	app.Get("/swagger", func(c fiber.Ctx) error {
		htmlData, err := os.ReadFile("src/static/swagger-ui.html")
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Swagger UI not found",
			})
		}
		c.Set("Content-Type", "text/html")
		return c.Send(htmlData)
	})
	
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON((fiber.Map{
			"message": "Hello, World!",
			"status":  fiber.StatusOK,
		}))
	})
	routes.SetupRoutes(app)
	log.Fatal(app.Listen(":3001"))
}
