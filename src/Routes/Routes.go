package routes

import (
	"github.com/gofiber/fiber/v3"
	controllers "github.com/tjens23/tabsplit-backend/src/Controllers"
	"github.com/tjens23/tabsplit-backend/src/middleware"
)

func SetupRoutes(app *fiber.App) {
	// User routes
	app.Get("/users", controllers.GetUsers)
	app.Patch("/users/update/:id", controllers.UpdateUser)
	app.Delete("/users/delete/:id", controllers.DeleteUser)

	// Auth routes
	app.Post("/auth/login", controllers.Login)
	app.Post("/auth/register", controllers.CreateUser)
	app.Post("/auth/logout", controllers.Logout)
	app.Post("/auth/refresh", controllers.RefreshToken)
	app.Get("/auth/user", middleware.IsAuth, controllers.GetUser)

	// Group routes
	app.Get("/groups", middleware.IsAuth, controllers.GetGroups)
	app.Get("/groups/:id", middleware.IsAuth, controllers.GetGroup)
	app.Post("/groups", middleware.IsAuth, controllers.CreateGroup)
	app.Patch("/groups/update/:id", middleware.IsAuth, controllers.UpdateGroup)
	app.Delete("/groups/delete/:id", middleware.IsAuth, controllers.DeleteGroup)

	// Expense routes
	app.Post("/expenses", middleware.IsAuth, controllers.CreateExpense)
	app.Get("/expenses", middleware.IsAuth, controllers.GetExpenses)
	app.Get("/expenses/:id", middleware.IsAuth, controllers.GetExpense)
	app.Patch("/expenses/update/:id", middleware.IsAuth, controllers.UpdateExpense)
	app.Delete("/expenses/delete/:id", middleware.IsAuth, controllers.DeleteExpense)

	// Settlement routes
	app.Post("/settlements/calculate", middleware.IsAuth, controllers.CalculateSettlements)
	app.Post("/settlements/create", middleware.IsAuth, controllers.CreateSettlements)
	app.Get("/groups/:id/settlements", middleware.IsAuth, controllers.GetGroupSettlements)
	app.Post("/settlements/:id/confirm", middleware.IsAuth, controllers.ConfirmSettlement)
}
