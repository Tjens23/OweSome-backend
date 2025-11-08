package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const SECRET_KEY = "supersecretstring"

func IsAuth(ctx fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")
	
	if cookie == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized, please login first",
		})
	}

	_, err := jwt.ParseWithClaims(cookie, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SECRET_KEY), nil
	})

	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized, please login first",
		})
	}

	return ctx.Next()
}
