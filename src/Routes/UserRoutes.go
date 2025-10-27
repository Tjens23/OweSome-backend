package routes

import (
	"github.com/gofiber/fiber/v3"
	controllers "github.com/tjens23/tabsplit-backend/src/Controllers"
)

func SetupUserRoutes(app *fiber.App) {
	app.Get("/users", controllers.GetUsers)
	app.Post("/users", controllers.CreateUser)
	app.Patch("/users/update/:id", controllers.UpdateUser)
	app.Delete("/users/delete/:id", controllers.DeleteUser)
}