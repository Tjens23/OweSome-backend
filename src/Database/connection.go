package database

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	err := godotenv.Load(); if err != nil {
		panic("Error loading .env file")
	}
	connection, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	connection.AutoMigrate(models.User{})

	DB = connection
	return DB
}