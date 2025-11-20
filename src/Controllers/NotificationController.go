package controllers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
)

// @Summary Get New Notifications
// @Description Gets new notification of the authenticated user
// @Tags notification
// @Accept json
// @Produce json
// @Router /notification [get]
func GetNewNotifications(ctx fiber.Ctx) error {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	var notifications []models.Notification
	if err := database.DB.Where("user_id = ? AND new = ? AND created_at >= ?", userID, true, time.Now().Add(-1*time.Minute)).
		Find(&notifications).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch groups: " + err.Error(),
		})
	}

	// Mark notifications as read
	if len(notifications) > 0 {
		if err := database.DB.
			Model(&models.Notification{}).
			Where("user_id = ? AND new = ?", userID, true).
			Update("new", false).Error; err != nil {

			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update notifications: " + err.Error(),
			})
		}
	}

	return ctx.Status(fiber.StatusOK).JSON(notifications)
}
