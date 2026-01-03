package controllers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// @Summary Get all users
// @Description Get list of all users
// @Tags users
// @Produce json
// @Success 200 {array} models.User "List of users"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users [get]
func GetUsers(c fiber.Ctx) error {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(users)
}

func GetUserByUsername(c fiber.Ctx) error {

	username := c.Params("username")
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(user)
}

// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.User true "User data"
// @Success 201 {object} models.User "User created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users [post]
func CreateUser(c fiber.Ctx) error {
	type RegisterInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}

	input := new(RegisterInput)

	if err := json.Unmarshal(c.Body(), input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	user := models.User{
		Username: input.Username,
		Password: input.Password,
		Email:    input.Email,
		Phone:    input.Phone,
	}

	PasswordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}
	user.Password = string(PasswordHash)

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// @Summary Update user profile
// @Description Update user profile information (username, email, phone) without requiring password
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body object true "User profile data (username, email, phone)"
// @Success 200 {object} models.User "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/update/:id [patch]
func UpdateUser(c fiber.Ctx) error {
	type UpdateInput struct {
		Username *string `json:"username"`
		Email    *string `json:"email"`
		Phone    *string `json:"phone"`
	}

	id := c.Params("id")
	user := new(models.User)

	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	input := new(UpdateInput)
	if err := json.Unmarshal(c.Body(), input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Only update fields that are provided
	if input.Username != nil {
		user.Username = *input.Username
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.Phone != nil {
		user.Phone = *input.Phone
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// @Summary Change user password
// @Description Change user password with old password verification
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param passwords body object true "Old and new password"
// @Success 200 {object} map[string]interface{} "Password changed successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized - incorrect old password"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/:id/change-password [patch]
func ChangePassword(c fiber.Ctx) error {
	type PasswordInput struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	id := c.Params("id")
	user := new(models.User)

	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	input := new(PasswordInput)
	if err := json.Unmarshal(c.Body(), input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Incorrect old password",
		})
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	user.Password = string(newPasswordHash)
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

func DeleteUser(c fiber.Ctx) error {
	id := c.Params("id")
	user := new(models.User)
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	if err := database.DB.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{})
}
