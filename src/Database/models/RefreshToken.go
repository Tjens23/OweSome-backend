package models

import "time"

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	Token     string    `gorm:"not null;unique"`
	UserID    uint      `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	IsRevoked bool      `gorm:"default:false"`
	
	User      User      `gorm:"foreignKey:UserID"`
}