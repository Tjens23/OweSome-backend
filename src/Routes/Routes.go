package routes

import (
	"github.com/gofiber/fiber/v3"
	controllers "github.com/tjens23/tabsplit-backend/src/Controllers"
	"github.com/tjens23/tabsplit-backend/src/middleware"
)

func SetupRoutes(app *fiber.App) {
	app.Get("/users", controllers.GetUsers)
	app.Post("/users", controllers.CreateUser)
	app.Patch("/users/update/:id", controllers.UpdateUser)
	app.Delete("/users/delete/:id", controllers.DeleteUser)
	app.Post("/auth/login", controllers.Login)
	app.Post("/auth/logout", controllers.Logout)
	app.Get("/auth/user", middleware.IsAuth, controllers.GetUser)
}
