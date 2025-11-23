package controllers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	database "github.com/tjens23/tabsplit-backend/src/Database"
	"github.com/tjens23/tabsplit-backend/src/Database/models"
	"gorm.io/gorm"
)

type CreateGroupInput struct {
	Name         string `json:"name"`
	ProfileImage string `json:"profile_image"`
	Description  string `json:"description"`
}

type UpdateGroupInput struct {
	Name         string `json:"name"`
	ProfileImage string `json:"profile_image"`
	Description  string `json:"description"`
}

// Helper function to extract user ID from JWT token
func getUserIDFromJWT(ctx fiber.Ctx) (uint, error) {
	cookie := ctx.Cookies("jwt")

	token, err := jwt.ParseWithClaims(cookie, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("supersecretstring"), nil
	})

	if err != nil {
		return 0, err
	}

	claims := token.Claims.(*jwt.RegisteredClaims)
	userID, err := strconv.Atoi(claims.Issuer)
	if err != nil {
		return 0, err
	}

	return uint(userID), nil
}

// @Summary Create a new group
// @Description Create a new expense group with the authenticated user as admin
// @Tags groups
// @Accept json
// @Produce json
// @Param group body CreateGroupInput true "Group data"
// @Success 201 {object} models.Group "Group created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Security ApiKeyAuth
// @Router /groups [post]
func CreateGroup(ctx fiber.Ctx) error {
	input := new(CreateGroupInput)

	if err := json.Unmarshal(ctx.Body(), input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON format. Please check your JSON syntax (missing commas, colons, etc.): " + err.Error(),
			"example": map[string]interface{}{
				"name":          "My Group Name",
				"description":   "Group description",
				"profile_image": "https://example.com/image.jpg",
			},
		})
	}

	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	group := models.Group{
		Name:         input.Name,
		ProfileImage: input.ProfileImage,
		Description:  input.Description,
		AdminID:      userID,
	}

	if err := database.DB.Create(&group).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create group: " + err.Error(),
		})
	}

	groupMember := models.GroupMember{
		GroupID: group.ID,
		UserID:  userID,
	}

	if err := database.DB.Create(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add user to group: " + err.Error(),
		})
	}

	// Load the group with admin info
	database.DB.Preload("GroupAdmin").First(&group, group.ID)

	type GroupWithBalance struct {
		ID           uint      `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		ProfileImage string    `json:"profile_image"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Admin        any       `json:"admin"`
		Members      any       `json:"members"`
		Status       float64   `json:"status"`
		Expenses     any       `json:"expenses"`
		Settlements  any       `json:"settlements"`
	}

	response := GroupWithBalance{
		ID:           group.ID,
		Name:         group.Name,
		Description:  group.Description,
		ProfileImage: group.ProfileImage,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
		Admin:        group.GroupAdmin,
		Members:      []models.User{},
		Status:       0,
		Expenses:     []models.Expense{},
		Settlements:  []models.Settlement{},
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Group created successfully",
		"group":   response,
	})
}

// @Summary Get user groups
// @Description Get all groups where the authenticated user is a member
// @Tags groups
// @Produce json
// @Success 200 {array} models.Group "List of user groups"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Security ApiKeyAuth
// @Router /groups [get]
func GetGroups(ctx fiber.Ctx) error {
	// Get user ID from JWT token (IsAuth middleware ensures token is valid)
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	var groupMemberships []models.GroupMember
	if err := database.DB.Where("user_id = ? AND is_active = ?", userID, true).
		Preload("Group.GroupAdmin").
		Joins("JOIN groups ON groups.id = group_members.group_id").
		Order("groups.updated_at DESC").
		Find(&groupMemberships).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch groups: " + err.Error(),
		})
	}

	type CompactGroup struct {
		ID           uint      `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		ProfileImage string    `json:"profile_image"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Status       float64   `json:"status"`
	}

	var groups []CompactGroup

	for _, membership := range groupMemberships {
		group := membership.Group

		// Calculate net balance for the user in this group
		var totalPaid float64
		var totalOwed float64

		// Load all expenses for this group with shares
		var expenses []models.Expense
		if err := database.DB.
			Preload("ExpenseShares").
			Where("group_id = ?", group.ID).
			Find(&expenses).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch expenses: " + err.Error(),
			})
		}

		for _, expense := range expenses {
			if expense.Settled {
				continue
			}

			// If user paid this expense, add to totalPaid
			if expense.PaidByID == userID {
				totalPaid += expense.Amount
			}

			// Check how much this user owes in this expense
			for _, share := range expense.ExpenseShares {
				if share.UserID == userID {
					totalOwed += share.AmountOwed
				}
			}
		}

		netBalance := totalPaid - totalOwed // positive = user is owed, negative = user owes

		groups = append(groups, CompactGroup{
			ID:           group.ID,
			Name:         group.Name,
			Description:  group.Description,
			ProfileImage: group.ProfileImage,
			CreatedAt:    group.CreatedAt,
			UpdatedAt:    group.UpdatedAt,
			Status:       netBalance,
		})
	}

	return ctx.JSON(fiber.Map{
		"groups": groups,
	})
}

// GetGroup returns a specific group by ID
func GetGroup(ctx fiber.Ctx) error {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	groupID := ctx.Params("id")

	var member models.GroupMember
	if err := database.DB.
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {

		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not a member of this group",
		})
	}

	var group models.Group
	if err := database.DB.Preload("GroupAdmin").
		Preload("Settlements").
		Preload("Settlements.Payer").
		Preload("Settlements.Receiver").
		Preload("Members.User").
		First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group: " + err.Error(),
		})
	}

	// Calculate net balance for this user in this group
	var expenses []models.Expense = []models.Expense{}
	if err := database.DB.
		Preload("PaidBy").
		Preload("ExpenseShares").
		Preload("ExpenseShares.User").
		Where("group_id = ?", group.ID).
		Find(&expenses).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expenses: " + err.Error(),
		})
	}

	var users []models.User
	for _, member := range group.Members {
		users = append(users, member.User)
	}

	var totalPaid float64
	var totalOwed float64

	for i := range expenses {
		expense := &expenses[i]

		if expense.Settled {
			continue
		}

		if expense.PaidByID == userID {
			totalPaid += expense.Amount
			expense.Status = expense.Amount
		}

		for _, share := range expense.ExpenseShares {
			if share.UserID == userID {
				totalOwed += share.AmountOwed
				expense.Status -= share.AmountOwed
			}
		}
	}

	netBalance := totalPaid - totalOwed

	// Build a response struct with net balance
	type GroupWithBalance struct {
		ID           uint      `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		ProfileImage string    `json:"profile_image"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Admin        any       `json:"admin"`
		Members      any       `json:"members"`
		Status       float64   `json:"status"`
		Expenses     any       `json:"expenses"`
		Settlements  any       `json:"settlements"`
	}

	response := GroupWithBalance{
		ID:           group.ID,
		Name:         group.Name,
		Description:  group.Description,
		ProfileImage: group.ProfileImage,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
		Admin:        group.GroupAdmin,
		Members:      users,
		Status:       netBalance,
		Expenses:     expenses,
		Settlements:  group.Settlements,
	}

	return ctx.JSON(response)
}

// @Summary Update a group
// @Description Update group information (only admin can update)
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body UpdateGroupInput true "Updated group data"
// @Success 200 {object} models.Group "Group updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden - Only admin can update"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Security ApiKeyAuth
// @Router /groups/update/{id} [patch]
func UpdateGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")
	input := new(UpdateGroupInput)

	// Use manual JSON parsing to fix parsing issues
	if err := json.Unmarshal(ctx.Body(), input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	var group models.Group
	if err := database.DB.
		Preload("Members.User").
		Preload("Settlements").
		Preload("Settlements.Payer").
		Preload("Settlements.Receiver").
		First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group: " + err.Error(),
		})
	}

	if group.AdminID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only group admin can update the group",
		})
	}

	group.Name = input.Name
	group.ProfileImage = input.ProfileImage
	group.Description = input.Description

	if err := database.DB.Save(&group).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update group: " + err.Error(),
		})
	}

	// Calculate net balance for this user in this group
	var expenses []models.Expense = []models.Expense{}
	if err := database.DB.
		Preload("PaidBy").
		Preload("ExpenseShares").
		Preload("ExpenseShares.User").
		Where("group_id = ?", group.ID).
		Find(&expenses).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expenses: " + err.Error(),
		})
	}

	var users []models.User
	for _, member := range group.Members {
		users = append(users, member.User)
	}

	var totalPaid float64
	var totalOwed float64

	for i := range expenses {
		expense := &expenses[i]

		if expense.Settled {
			continue
		}

		if expense.PaidByID == userID {
			totalPaid += expense.Amount
			expense.Status = expense.Amount
		}

		for _, share := range expense.ExpenseShares {
			if share.UserID == userID {
				totalOwed += share.AmountOwed
				expense.Status = -share.AmountOwed
			}
		}
	}

	netBalance := totalPaid - totalOwed

	// Build a response struct with net balance
	type GroupWithBalance struct {
		ID           uint      `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		ProfileImage string    `json:"profile_image"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Admin        any       `json:"admin"`
		Members      any       `json:"members"`
		Status       float64   `json:"status"`
		Expenses     any       `json:"expenses"`
		Settlements  any       `json:"settlements"`
	}

	response := GroupWithBalance{
		ID:           group.ID,
		Name:         group.Name,
		Description:  group.Description,
		ProfileImage: group.ProfileImage,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
		Admin:        group.GroupAdmin,
		Members:      users,
		Status:       netBalance,
		Expenses:     expenses,
		Settlements:  group.Settlements,
	}

	return ctx.JSON(fiber.Map{
		"message": "Group updated successfully",
		"group":   response,
	})
}

// @Summary Delete a group
// @Description Delete a group and all associated data (only admin can delete)
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} map[string]interface{} "Group deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden - Only admin can delete"
// @Failure 404 {object} map[string]interface{} "Group not found"
// @Security ApiKeyAuth
// @Router /groups/delete/{id} [delete]
func DeleteGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")

	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to extract user ID from token",
		})
	}

	var group models.Group
	if err := database.DB.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group: " + err.Error(),
		})
	}

	if group.AdminID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only group admin can delete the group",
		})
	}

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

	// Delete all group members first
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
		tx.Rollback()
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete group members: " + err.Error(),
		})
	}

	// Delete all expense shares for expenses in this group first (if expense_shares table exists)
	if tx.Migrator().HasTable(&models.ExpenseShare{}) {
		var expenseIDs []uint
		if err := tx.Model(&models.Expense{}).Where("group_id = ?", groupID).Pluck("id", &expenseIDs).Error; err != nil {
			tx.Rollback()
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get expense IDs: " + err.Error(),
			})
		}

		if len(expenseIDs) > 0 {
			if err := tx.Where("expense_id IN ?", expenseIDs).Delete(&models.ExpenseShare{}).Error; err != nil {
				tx.Rollback()
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to delete expense shares: " + err.Error(),
				})
			}
		}
	}

	// Delete all expenses associated with this group
	if err := tx.Where("group_id = ?", groupID).Delete(&models.Expense{}).Error; err != nil {
		tx.Rollback()
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete group expenses: " + err.Error(),
		})
	}

	// Delete all settlements associated with this group (if settlements exist)
	// Check if settlements table exists first
	if tx.Migrator().HasTable(&models.Settlement{}) {
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Settlement{}).Error; err != nil {
			tx.Rollback()
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete group settlements: " + err.Error(),
			})
		}
	}

	// Finally delete the group itself
	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete group: " + err.Error(),
		})
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group and all associated data deleted successfully",
	})
}

func AddMemberToGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")

	var input struct {
		UserID uint `json:"user_id"`
	}

	if err := ctx.Bind().JSON(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	var group models.Group
	if err := database.DB.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group: " + err.Error(),
		})
	}

	var user models.User
	if err := database.DB.First(&user, input.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user: " + err.Error(),
		})
	}

	var existingMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, input.UserID).First(&existingMember).Error; err == nil {
		if existingMember.IsActive {
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "User is already a member of this group",
			})
		} else {
			// Reactivate membership
			existingMember.IsActive = true
			database.DB.Save(&existingMember)
			return ctx.JSON(fiber.Map{
				"message": "User added to group successfully",
			})
		}
	}

	groupMember := models.GroupMember{
		GroupID: group.ID,
		UserID:  input.UserID,
	}

	if err := database.DB.Create(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add user to group: " + err.Error(),
		})
	}

	if err := database.DB.Create(&models.Notification{
		Message: "You have been added to a new group: " + group.Name,
		UserID:  input.UserID,
		New:     true,
	}).Error; err != nil {
		println("Could not send notification " + err.Error())
	}

	return ctx.JSON(fiber.Map{
		"message": "User added to group successfully",
	})
}

func RemoveMemberFromGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")

	var input struct {
		UserID uint `json:"user_id"`
	}

	if err := ctx.Bind().JSON(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	var groupMember models.GroupMember
	if err := database.DB.Preload("Group").Where("group_id = ? AND user_id = ? AND is_active = ?", groupID, input.UserID, true).First(&groupMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User is not a member of this group",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group membership: " + err.Error(),
		})
	}

	if err := database.DB.Delete(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove user from group: " + err.Error(),
		})
	}

	if err := database.DB.Create(&models.Notification{
		Message: "You have been removed from group: " + groupMember.Group.Name,
		UserID:  input.UserID,
		New:     true,
	}).Error; err != nil {
		println("Could not send notification " + err.Error())
	}

	return ctx.JSON(fiber.Map{
		"message": "User removed from group successfully",
	})
}
