package models

import "time"

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	Email    string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Phone    string `gorm:"unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	

	AdminGroups []Group `gorm:"foreignKey:AdminID"`
	

	GroupMemberships []GroupMember `gorm:"foreignKey:UserID"`
	
	PaidExpenses []Expense `gorm:"foreignKey:PaidByID"`
	
	ExpenseShares []ExpenseShare `gorm:"foreignKey:UserID"`
	
	PaymentsMade []Settlement `gorm:"foreignKey:PayerID"`
	
	PaymentsReceived []Settlement `gorm:"foreignKey:ReceiverID"`
}
