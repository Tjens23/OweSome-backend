package models

import "time"

type Expense struct {
	ID          uint      `gorm:"primaryKey"`
	Amount      float64   `gorm:"not null"`
	Description string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	GroupID  uint `gorm:"not null"`
	PaidByID uint `gorm:"not null"`

	Group         Group          `gorm:"foreignKey:GroupID" json:"-"`
	PaidBy        User           `gorm:"foreignKey:PaidByID"`
	ExpenseShares []ExpenseShare `gorm:"foreignKey:ExpenseID"`

	Status float64 `gorm:"-"`
}

type ExpenseShare struct {
	ID         uint    `gorm:"primaryKey"`
	ExpenseID  uint    `gorm:"not null"`
	UserID     uint    `gorm:"not null"`
	AmountOwed float64 `gorm:"not null"`
	IsPaid     bool    `gorm:"default:false"`

	Expense Expense `gorm:"foreignKey:ExpenseID" json:"-"`
	User    User    `gorm:"foreignKey:UserID"`
}

type Settlement struct {
	ID          uint      `gorm:"primaryKey"`
	Amount      float64   `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	IsConfirmed bool      `gorm:"default:false"`

	GroupID    uint `gorm:"not null"`
	PayerID    uint `gorm:"not null"`
	ReceiverID uint `gorm:"not null"`

	Group    Group `gorm:"foreignKey:GroupID" json:"-"`
	Payer    User  `gorm:"foreignKey:PayerID"`
	Receiver User  `gorm:"foreignKey:ReceiverID"`
}
