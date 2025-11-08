package controllers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)


type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// @Summary User login
// @Description Authenticate user with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginInput true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Router /auth/login [post]
func Login(ctx fiber.Ctx) error {
	input := new(LoginInput)
	
	if err := json.Unmarshal(ctx.Body(), input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}
	var user models.User

	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid username or password",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "something went wrong" + err.Error(),
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Incorrect password",
		})
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: strconv.Itoa(int(user.ID)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	token, err := claims.SignedString([]byte("supersecretstring"))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token: " + err.Error(),
		})
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}
	ctx.Cookie(&cookie)
	return ctx.JSON(fiber.Map{
		"message": "Welcome back, "  + user.Username,
	})
}

// @Summary User logout
// @Description Logout user by clearing JWT cookie
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Router /auth/logout [post]
func Logout(ctx fiber.Ctx) error {
	
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}
	ctx.Cookie(&cookie)
	return ctx.JSON(fiber.Map{
		"message": "Logout successful",
	})
}

// @Summary Get current user
// @Description Get current authenticated user information
// @Tags auth
// @Produce json
// @Success 200 {object} models.User "User information"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Security ApiKeyAuth
// @Router /auth/user [get]
func GetUser(ctx fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")
	
	token, err := jwt.ParseWithClaims(cookie, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("supersecretstring"), nil
	})

	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	claims := token.Claims.(*jwt.RegisteredClaims)
	userID, _ := strconv.Atoi(claims.Issuer)

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return ctx.JSON(user)
}
