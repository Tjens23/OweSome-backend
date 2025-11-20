package models

import "time"

type Notification struct {
	ID        uint      `gorm:"primaryKey"`
	Message   string    `gorm:"not null"`
	UserID    uint      `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	New       bool      `gorm:"not null"`
}
