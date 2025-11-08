package models

import "time"

type GroupMember struct {
	ID        uint      `gorm:"primaryKey"`
	GroupID   uint      `gorm:"not null"`
	UserID    uint      `gorm:"not null"`
	JoinedAt  time.Time `gorm:"autoCreateTime"`
	IsActive  bool      `gorm:"default:true"`
	
	Group     Group     `gorm:"foreignKey:GroupID"`
	User      User      `gorm:"foreignKey:UserID"`
}