package controllers

import (
	"crypto/rand"
	"encoding/base64"
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

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateAccessToken(userID uint) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    strconv.Itoa(int(userID)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 hours
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	return claims.SignedString([]byte("supersecretstring"))
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

	accessToken, err := generateAccessToken(user.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token: " + err.Error(),
		})
	}

	refreshTokenString, err := generateRefreshToken()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate refresh token: " + err.Error(),
		})
	}

	// Revoke any existing refresh tokens for this user
	database.DB.Model(&models.RefreshToken{}).Where("user_id = ?", user.ID).Update("is_revoked", true)

	refreshToken := models.RefreshToken{
		Token:     refreshTokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		IsRevoked: false,
	}
	if err := database.DB.Create(&refreshToken).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save refresh token: " + err.Error(),
		})
	}

	accessCookie := fiber.Cookie{
		Name:     "jwt",
		Value:    accessToken,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}
	ctx.Cookie(&accessCookie)

	refreshCookie := fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
	}
	ctx.Cookie(&refreshCookie)

	return ctx.JSON(fiber.Map{
		"message":       "Welcome back, " + user.Username,
		"access_token":  accessToken,
		"refresh_token": refreshTokenString,
		"expires_in":    time.Now().Add(24 * time.Hour).Unix(), 
	})
}

// @Summary User logout
// @Description Logout user by clearing JWT cookie
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Router /auth/logout [post]
func Logout(ctx fiber.Ctx) error {
	// Get user ID from JWT to revoke refresh tokens
	jwtCookie := ctx.Cookies("jwt")
	if jwtCookie != "" {
		token, err := jwt.ParseWithClaims(jwtCookie, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte("supersecretstring"), nil
		})
		
		if err == nil {
			claims := token.Claims.(*jwt.RegisteredClaims)
			userID, _ := strconv.Atoi(claims.Issuer)
			
			// Revoke all refresh tokens for this user
			database.DB.Model(&models.RefreshToken{}).Where("user_id = ?", userID).Update("is_revoked", true)
		}
	}

	accessCookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}
	ctx.Cookie(&accessCookie)

	refreshCookie := fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}
	ctx.Cookie(&refreshCookie)

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

// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body RefreshInput true "Refresh token"
// @Success 200 {object} map[string]interface{} "New tokens generated"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func RefreshToken(ctx fiber.Ctx) error {
	var input RefreshInput
	
	// Try to get refresh token from request body
	if err := json.Unmarshal(ctx.Body(), &input); err != nil {
		// If not in body, try to get from cookie
		input.RefreshToken = ctx.Cookies("refresh_token")
		if input.RefreshToken == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Refresh token required in body or cookie",
			})
		}
	}

	// Find refresh token in database
	var refreshToken models.RefreshToken
	if err := database.DB.Where("token = ? AND is_revoked = false", input.RefreshToken).First(&refreshToken).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	// Check if refresh token is expired
	if refreshToken.ExpiresAt.Before(time.Now()) {
		// Mark token as revoked if expired
		database.DB.Model(&refreshToken).Update("is_revoked", true)
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Refresh token expired",
		})
	}

	// Get user
	var user models.User
	if err := database.DB.Where("id = ?", refreshToken.UserID).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Generate new access token
	newAccessToken, err := generateAccessToken(user.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate new access token: " + err.Error(),
		})
	}

	// Generate new refresh token
	newRefreshTokenString, err := generateRefreshToken()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate new refresh token: " + err.Error(),
		})
	}

	// Revoke old refresh token
	database.DB.Model(&refreshToken).Update("is_revoked", true)

	// Save new refresh token to database
	newRefreshToken := models.RefreshToken{
		Token:     newRefreshTokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		IsRevoked: false,
	}
	if err := database.DB.Create(&newRefreshToken).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save new refresh token: " + err.Error(),
		})
	}

	// Set new access token in cookie
	accessCookie := fiber.Cookie{
		Name:     "jwt",
		Value:    newAccessToken,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}
	ctx.Cookie(&accessCookie)

	// Set new refresh token in cookie
	refreshCookie := fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshTokenString,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
	}
	ctx.Cookie(&refreshCookie)

	return ctx.JSON(fiber.Map{
		"message":       "Tokens refreshed successfully",
		"access_token":  newAccessToken,
		"refresh_token": newRefreshTokenString,
		"expires_in":    86400, // 24 hours in seconds
	})
}
