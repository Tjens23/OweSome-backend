package controllers

import (
	"encoding/json"
	"strconv"

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

// CreateGroup creates a new group with the authenticated user as admin
func CreateGroup(ctx fiber.Ctx) error {
	input := new(CreateGroupInput)
	
	// Use manual JSON parsing to fix parsing issues
	if err := json.Unmarshal(ctx.Body(), input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON format. Please check your JSON syntax (missing commas, colons, etc.): " + err.Error(),
			"example": map[string]interface{}{
				"name": "My Group Name",
				"description": "Group description", 
				"profile_image": "https://example.com/image.jpg",
			},
		})
	}

	// Get user ID from JWT token (IsAuth middleware ensures token is valid)
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

	// Automatically add the creator as a member
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

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Group created successfully",
		"group":   group,
	})
}

// GetGroups returns all groups for the authenticated user
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
		Find(&groupMemberships).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch groups: " + err.Error(),
		})
	}

	var groups []models.Group
	for _, membership := range groupMemberships {
		groups = append(groups, membership.Group)
	}

	return ctx.JSON(fiber.Map{
		"groups": groups,
	})
}

// GetGroup returns a specific group by ID
func GetGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")
	
	var group models.Group
	if err := database.DB.Preload("GroupAdmin").
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

	return ctx.JSON(group)
}

// UpdateGroup updates a group (only admin can update)
func UpdateGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")
	input := new(UpdateGroupInput)
	
	// Use manual JSON parsing to fix parsing issues
	if err := json.Unmarshal(ctx.Body(), input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	// Get user ID from JWT token (IsAuth middleware ensures token is valid)
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

	// Check if user is admin
	if group.AdminID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only group admin can update the group",
		})
	}

	// Update fields
	group.Name = input.Name
	group.ProfileImage = input.ProfileImage
	group.Description = input.Description

	if err := database.DB.Save(&group).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update group: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group updated successfully",
		"group":   group,
	})
}

// DeleteGroup deletes a group (only admin can delete)
func DeleteGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")

	// Get user ID from JWT token (IsAuth middleware ensures token is valid)
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

	// Check if user is admin
	if group.AdminID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only group admin can delete the group",
		})
	}

	// Start a transaction to ensure data consistency
	tx := database.DB.Begin()
	if tx.Error != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction: " + tx.Error.Error(),
		})
	}

	// Rollback transaction on error
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

// AddMemberToGroup adds a user to a group
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

	// Check if group exists
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

	// Check if user exists
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

	// Check if already a member
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

	// Add new member
	groupMember := models.GroupMember{
		GroupID: group.ID,
		UserID:  input.UserID,
	}

	if err := database.DB.Create(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add user to group: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "User added to group successfully",
	})
}

// RemoveMemberFromGroup removes a user from a group
func RemoveMemberFromGroup(ctx fiber.Ctx) error {
	groupID := ctx.Params("id")
	userIDToRemove := ctx.Params("userId")

	var groupMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND is_active = ?", groupID, userIDToRemove, true).First(&groupMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User is not a member of this group",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch group membership: " + err.Error(),
		})
	}

	// Soft delete by setting IsActive to false
	groupMember.IsActive = false
	if err := database.DB.Save(&groupMember).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove user from group: " + err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "User removed from group successfully",
	})
}