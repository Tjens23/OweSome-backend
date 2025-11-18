package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"unique;not null"`
	Email     string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null" json:"-"`
	Phone     string    `gorm:"unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	AdminGroups []Group `gorm:"foreignKey:AdminID"`

	GroupMemberships []GroupMember `gorm:"foreignKey:UserID" json:"-"`

	PaidExpenses []Expense `gorm:"foreignKey:PaidByID" json:"-"`

	ExpenseShares []ExpenseShare `gorm:"foreignKey:UserID" json:"-"`

	PaymentsMade []Settlement `gorm:"foreignKey:PayerID" json:"-"`

	PaymentsReceived []Settlement `gorm:"foreignKey:ReceiverID" json:"-"`
}
