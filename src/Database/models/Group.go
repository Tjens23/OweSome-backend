package models

import "time"

type Group struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	ProfileImage string `gorm:"not null"`
	Description  string `gorm:"not null"`
	AdminID      uint   `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	
	
	GroupAdmin   User   `gorm:"foreignKey:AdminID"`
	
	Members      []GroupMember `gorm:"foreignKey:GroupID"`
	
	Expenses     []Expense `gorm:"foreignKey:GroupID"`
	
	Settlements  []Settlement `gorm:"foreignKey:GroupID"`
}