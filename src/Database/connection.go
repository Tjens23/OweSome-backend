package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	_ = godotenv.Load() 

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	const maxRetries = 30
	retryDelay := 1 * time.Second

	var (
		db  *gorm.DB
		err error
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, _ := db.DB()
			if pingErr := sqlDB.Ping(); pingErr == nil {
				log.Println("Database connected successfully!")
				goto connected
			}
			err = fmt.Errorf("ping failed")
		}

		if attempt < maxRetries {
			log.Printf(
				"Database attempt %d/%d failed (%v). Retrying in %s...",
				attempt, maxRetries, err, retryDelay,
			)
			time.Sleep(retryDelay)
			if retryDelay < 5*time.Second {
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
			}
			continue
		}

		log.Fatalf("Failed to connect after %d attempts: %v", maxRetries, err)
	}

connected:
	if migrateErr := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Expense{},
		&models.GroupMember{},
		&models.ExpenseShare{},
		&models.Settlement{},
		&models.RefreshToken{},
	); migrateErr != nil {
		log.Fatalf("AutoMigrate failed: %v", migrateErr)
	}

	DB = db
	return DB
}
