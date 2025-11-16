package controllers

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/gofiber/fiber/v3"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"gorm.io/gorm"
)

// SettlementInput represents the input for settlement calculation
type SettlementInput struct {
	GroupID uint `json:"group_id"`
}

// DebtBalance represents the net balance for a user (positive = owed to them, negative = they owe)
type DebtBalance struct {
	UserID uint    `json:"user_id"`
	Amount float64 `json:"amount"`
}

// SettlementTransaction represents a transaction needed to settle debts
type SettlementTransaction struct {
	PayerID    uint    `json:"payer_id"`
	ReceiverID uint    `json:"receiver_id"`
	Amount     float64 `json:"amount"`
}

// CalculateSettlements calculates optimal settlements for a group
// @Summary Calculate settlements for a group
// @Description Calculate the minimum number of transactions needed to settle all debts in a group
// @Tags settlements
// @Accept json
// @Produce json
// @Param settlement body SettlementInput true "Settlement data"
// @Success 200 {object} map[string]interface{} "Settlement calculations"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Security ApiKeyAuth
// @Router /settlements/calculate [post]
func CalculateSettlements(ctx fiber.Ctx) error {
	var input SettlementInput
	
	if err := json.Unmarshal(ctx.Body(), &input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	// Get user ID from JWT token
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	// Check if user is member of the group
	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", input.GroupID, userID).First(&groupMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not a member of this group",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check group membership: " + err.Error(),
		})
	}

	// Calculate debt balances
	balances, err := calculateDebtBalances(input.GroupID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to calculate balances: " + err.Error(),
		})
	}

	// Calculate optimal settlements
	settlements := calculateOptimalSettlements(balances)

	// Get user details for response
	settlementsWithUsers, err := enrichSettlementsWithUserDetails(settlements)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user details: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"group_id":    input.GroupID,
		"settlements": settlementsWithUsers,
		"message":     "Settlement calculation completed successfully",
	})
}

// CreateSettlements creates settlements in the database
// @Summary Create settlements for a group
// @Description Create settlement records based on calculated optimal settlements
// @Tags settlements
// @Accept json
// @Produce json
// @Param settlement body SettlementInput true "Settlement data"
// @Success 201 {object} map[string]interface{} "Settlements created"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Security ApiKeyAuth
// @Router /settlements/create [post]
func CreateSettlements(ctx fiber.Ctx) error {
	var input SettlementInput
	
	if err := json.Unmarshal(ctx.Body(), &input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	// Get user ID from JWT token
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	// Check if user is admin of the group
	var group models.Group
	if err := database.DB.Where("id = ? AND admin_id = ?", input.GroupID, userID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only group admin can create settlements",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify admin status: " + err.Error(),
		})
	}

	// Calculate debt balances
	balances, err := calculateDebtBalances(input.GroupID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to calculate balances: " + err.Error(),
		})
	}

	// Calculate optimal settlements
	settlements := calculateOptimalSettlements(balances)

	// Start transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction: " + tx.Error.Error(),
		})
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create settlement records
	var createdSettlements []models.Settlement
	for _, settlement := range settlements {
		settlementRecord := models.Settlement{
			GroupID:     input.GroupID,
			PayerID:     settlement.PayerID,
			ReceiverID:  settlement.ReceiverID,
			Amount:      settlement.Amount,
			IsConfirmed: false,
		}

		if err := tx.Create(&settlementRecord).Error; err != nil {
			tx.Rollback()
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create settlement: " + err.Error(),
			})
		}

		createdSettlements = append(createdSettlements, settlementRecord)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction: " + err.Error(),
		})
	}

	// Get user details for response
	settlementsWithUsers, err := enrichSettlementsWithUserDetails(settlements)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user details: " + err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"group_id":    input.GroupID,
		"settlements": settlementsWithUsers,
		"message":     "Settlements created successfully",
	})
}

// calculateDebtBalances calculates the net balance for each user in a group
func calculateDebtBalances(groupID uint) ([]DebtBalance, error) {
	// Get all expenses for the group
	var expenses []models.Expense
	if err := database.DB.Where("group_id = ?", groupID).Preload("ExpenseShares").Find(&expenses).Error; err != nil {
		return nil, err
	}

	// Calculate net balances for each user
	userBalances := make(map[uint]float64)

	for _, expense := range expenses {
		// Add the amount paid by the payer
		userBalances[expense.PaidByID] += expense.Amount

		// Subtract the amounts owed by each user
		for _, share := range expense.ExpenseShares {
			if !share.IsPaid {
				userBalances[share.UserID] -= share.AmountOwed
			}
		}
	}

	// Convert to slice and filter out zero balances
	var balances []DebtBalance
	for userID, amount := range userBalances {
		if math.Abs(amount) > 0.01 { // Ignore amounts less than 1 cent
			balances = append(balances, DebtBalance{
				UserID: userID,
				Amount: amount,
			})
		}
	}

	return balances, nil
}

// calculateOptimalSettlements implements the debt simplification algorithm
func calculateOptimalSettlements(balances []DebtBalance) []SettlementTransaction {
	if len(balances) == 0 {
		return []SettlementTransaction{}
	}

	// Make a copy to avoid modifying original
	workingBalances := make([]DebtBalance, len(balances))
	copy(workingBalances, balances)

	var settlements []SettlementTransaction

	for {
		// Sort balances: most negative (owes most) first, then most positive (owed most) first
		sort.Slice(workingBalances, func(i, j int) bool {
			if workingBalances[i].Amount < 0 && workingBalances[j].Amount < 0 {
				return workingBalances[i].Amount < workingBalances[j].Amount // More negative first
			}
			if workingBalances[i].Amount > 0 && workingBalances[j].Amount > 0 {
				return workingBalances[i].Amount > workingBalances[j].Amount // More positive first
			}
			return workingBalances[i].Amount < workingBalances[j].Amount // Negative before positive
		})

		// Find the most negative and most positive balances
		var minIndex, maxIndex int = -1, -1
		
		for i, balance := range workingBalances {
			if balance.Amount < -0.01 && minIndex == -1 {
				minIndex = i
			}
			if balance.Amount > 0.01 && maxIndex == -1 {
				maxIndex = i
			}
		}

		// If no negative or no positive balance found, we're done
		if minIndex == -1 || maxIndex == -1 {
			break
		}

		debtor := &workingBalances[minIndex]
		creditor := &workingBalances[maxIndex]

		// Calculate settlement amount (minimum of what debtor owes and what creditor is owed)
		settlementAmount := math.Min(math.Abs(debtor.Amount), creditor.Amount)

		// Round to 2 decimal places
		settlementAmount = math.Round(settlementAmount*100) / 100

		// Create settlement transaction
		settlements = append(settlements, SettlementTransaction{
			PayerID:    debtor.UserID,
			ReceiverID: creditor.UserID,
			Amount:     settlementAmount,
		})

		// Update balances
		debtor.Amount += settlementAmount
		creditor.Amount -= settlementAmount

		// Remove balances that are now effectively zero
		workingBalances = filterNonZeroBalances(workingBalances)
	}

	return settlements
}

// filterNonZeroBalances removes balances that are effectively zero
func filterNonZeroBalances(balances []DebtBalance) []DebtBalance {
	var filtered []DebtBalance
	for _, balance := range balances {
		if math.Abs(balance.Amount) > 0.01 { // Keep balances greater than 1 cent
			filtered = append(filtered, balance)
		}
	}
	return filtered
}

// enrichSettlementsWithUserDetails adds user information to settlements
func enrichSettlementsWithUserDetails(settlements []SettlementTransaction) ([]map[string]interface{}, error) {
	var enrichedSettlements []map[string]interface{}

	for _, settlement := range settlements {
		var payer, receiver models.User
		
		if err := database.DB.First(&payer, settlement.PayerID).Error; err != nil {
			return nil, err
		}
		
		if err := database.DB.First(&receiver, settlement.ReceiverID).Error; err != nil {
			return nil, err
		}

		enrichedSettlement := map[string]interface{}{
			"payer": map[string]interface{}{
				"id":       payer.ID,
				"username": payer.Username,
				"email":    payer.Email,
			},
			"receiver": map[string]interface{}{
				"id":       receiver.ID,
				"username": receiver.Username,
				"email":    receiver.Email,
			},
			"amount": settlement.Amount,
		}

		enrichedSettlements = append(enrichedSettlements, enrichedSettlement)
	}

	return enrichedSettlements, nil
}

// GetGroupSettlements gets existing settlements for a group
// @Summary Get settlements for a group
// @Description Get all existing settlements for a specific group
// @Tags settlements
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {array} models.Settlement "List of settlements"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Not a group member"
// @Security ApiKeyAuth
// @Router /groups/{id}/settlements [get]
func GetGroupSettlements(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")
	
	// Get user ID from JWT token
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	// Check if user is member of the group
	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&groupMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not a member of this group",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check group membership: " + err.Error(),
		})
	}

	// Get settlements for the group
	var settlements []models.Settlement
	if err := database.DB.Where("group_id = ?", groupID).
		Preload("Payer").
		Preload("Receiver").
		Find(&settlements).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch settlements: " + err.Error(),
		})
	}

	return ctx.JSON(settlements)
}

// ConfirmSettlement confirms a settlement transaction
// @Summary Confirm a settlement
// @Description Mark a settlement as confirmed by the payer
// @Tags settlements
// @Produce json
// @Param id path string true "Settlement ID"
// @Success 200 {object} map[string]interface{} "Settlement confirmed"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Not authorized to confirm"
// @Failure 404 {object} map[string]interface{} "Settlement not found"
// @Security ApiKeyAuth
// @Router /settlements/{id}/confirm [post]
func ConfirmSettlement(ctx fiber.Ctx) error {
	settlementID := ctx.Params("id")
	
	// Get user ID from JWT token
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	// Get settlement
	var settlement models.Settlement
	if err := database.DB.First(&settlement, settlementID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Settlement not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch settlement: " + err.Error(),
		})
	}

	// Check if user is the payer of this settlement
	if settlement.PayerID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the payer can confirm this settlement",
		})
	}

	// Update settlement status
	if err := database.DB.Model(&settlement).Update("is_confirmed", true).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to confirm settlement: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Settlement confirmed successfully",
		"settlement_id": settlement.ID,
	})
}