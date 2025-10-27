package controllers

import (
	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
)

func GetUsers(c fiber.Ctx) error {
    var users []models.User
    if err := database.DB.Find(&users).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }
    return c.JSON(users)
}