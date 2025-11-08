package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"gorm.io/gorm"
)

type CreateExpenseInput struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	GroupID     uint    `json:"group_id"`
	SplitAmong  []uint  `json:"split_among"` // Array of user IDs to split among
}

type UpdateExpenseInput struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// CreateExpense creates a new expense and splits it among specified users
func CreateExpense(ctx fiber.Ctx) error {
	input := new(CreateExpenseInput)
	
	if err := ctx.Bind().JSON(input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	// Get current user from auth
	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Verify user is member of the group
	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND is_active = ?", input.GroupID, userID, true).First(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not a member of this group",
		})
	}

	// Create the expense
	expense := models.Expense{
		Amount:      input.Amount,
		Description: input.Description,
		GroupID:     input.GroupID,
		PaidByID:    uint(userID),
	}

	if err := database.DB.Create(&expense).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create expense: " + err.Error(),
		})
	}

	// Calculate split amount
	splitAmount := input.Amount / float64(len(input.SplitAmong))

	// Create expense shares
	for _, splitUserID := range input.SplitAmong {
		// Verify each user is a member of the group
		var memberCheck models.GroupMember
		if err := database.DB.Where("group_id = ? AND user_id = ? AND is_active = ?", input.GroupID, splitUserID, true).First(&memberCheck).Error; err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "User ID " + strconv.Itoa(int(splitUserID)) + " is not a member of this group",
			})
		}

		expenseShare := models.ExpenseShare{
			ExpenseID:  expense.ID,
			UserID:     splitUserID,
			AmountOwed: splitAmount,
		}

		if err := database.DB.Create(&expenseShare).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create expense share: " + err.Error(),
			})
		}
	}

	// Load expense with relationships
	database.DB.Preload("PaidBy").Preload("Group").Preload("ExpenseShares.User").First(&expense, expense.ID)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Expense created successfully",
		"expense": expense,
	})
}

// GetExpenses returns all expenses for a group
func GetExpenses(ctx fiber.Ctx) error {
	groupID := ctx.Query("group_id")
	if groupID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group ID is required",
		})
	}

	// Verify user is member of the group
	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND is_active = ?", groupID, userID, true).First(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not a member of this group",
		})
	}

	var expenses []models.Expense
	if err := database.DB.Where("group_id = ?", groupID).
		Preload("PaidBy").
		Preload("ExpenseShares.User").
		Order("created_at DESC").
		Find(&expenses).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expenses: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"expenses": expenses,
	})
}

// GetExpense returns a specific expense by ID
func GetExpense(ctx fiber.Ctx) error {
	expenseID := ctx.Params("id")

	var expense models.Expense
	if err := database.DB.Preload("PaidBy").
		Preload("Group").
		Preload("ExpenseShares.User").
		First(&expense, expenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Expense not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expense: " + err.Error(),
		})
	}

	// Verify user is member of the group
	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND is_active = ?", expense.GroupID, userID, true).First(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not a member of this group",
		})
	}

	return ctx.JSON(expense)
}

// UpdateExpense updates an expense (only the person who paid can update)
func UpdateExpense(ctx fiber.Ctx) error {
	expenseID := ctx.Params("id")
	input := new(UpdateExpenseInput)
	
	if err := ctx.Bind().JSON(input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var expense models.Expense
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Expense not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expense: " + err.Error(),
		})
	}

	// Check if user paid for this expense
	if expense.PaidByID != uint(userID) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the person who paid can update this expense",
		})
	}

	// Update expense
	expense.Amount = input.Amount
	expense.Description = input.Description

	if err := database.DB.Save(&expense).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update expense: " + err.Error(),
		})
	}

	// If amount changed, update expense shares proportionally
	var expenseShares []models.ExpenseShare
	if err := database.DB.Where("expense_id = ?", expense.ID).Find(&expenseShares).Error; err == nil {
		newSplitAmount := input.Amount / float64(len(expenseShares))
		for _, share := range expenseShares {
			share.AmountOwed = newSplitAmount
			database.DB.Save(&share)
		}
	}

	return ctx.JSON(fiber.Map{
		"message": "Expense updated successfully",
		"expense": expense,
	})
}

// DeleteExpense deletes an expense (only the person who paid can delete)
func DeleteExpense(ctx fiber.Ctx) error {
	expenseID := ctx.Params("id")

	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var expense models.Expense
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Expense not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expense: " + err.Error(),
		})
	}

	// Check if user paid for this expense
	if expense.PaidByID != uint(userID) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the person who paid can delete this expense",
		})
	}

	// Delete expense shares first (due to foreign key constraints)
	if err := database.DB.Where("expense_id = ?", expense.ID).Delete(&models.ExpenseShare{}).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete expense shares: " + err.Error(),
		})
	}

	// Delete expense
	if err := database.DB.Delete(&expense).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete expense: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Expense deleted successfully",
	})
}

// GetUserBalance calculates what a user owes or is owed in a group
func GetUserBalance(ctx fiber.Ctx) error {
	groupID := ctx.Query("group_id")
	if groupID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group ID is required",
		})
	}

	userIDStr := ctx.Get("X-User-ID")
	if userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found",
		})
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Calculate total paid by user
	var totalPaid float64
	database.DB.Model(&models.Expense{}).Where("group_id = ? AND paid_by_id = ?", groupID, userID).Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)

	// Calculate total owed by user
	var totalOwed float64
	database.DB.Table("expense_shares").
		Joins("JOIN expenses ON expenses.id = expense_shares.expense_id").
		Where("expenses.group_id = ? AND expense_shares.user_id = ?", groupID, userID).
		Select("COALESCE(SUM(expense_shares.amount_owed), 0)").
		Scan(&totalOwed)

	balance := totalPaid - totalOwed

	return ctx.JSON(fiber.Map{
		"user_id":     userID,
		"group_id":    groupID,
		"total_paid":  totalPaid,
		"total_owed":  totalOwed,
		"balance":     balance,
		"status":      func() string {
			if balance > 0 {
				return "owed" // Others owe this user
			} else if balance < 0 {
				return "owes" // This user owes others
			}
			return "settled"
		}(),
	})
}

// MarkExpenseSharePaid marks an expense share as paid
func MarkExpenseSharePaid(ctx fiber.Ctx) error {
	expenseShareID := ctx.Params("shareId")

	var expenseShare models.ExpenseShare
	if err := database.DB.First(&expenseShare, expenseShareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Expense share not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expense share: " + err.Error(),
		})
	}

	expenseShare.IsPaid = true
	if err := database.DB.Save(&expenseShare).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update expense share: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Expense share marked as paid",
	})
}