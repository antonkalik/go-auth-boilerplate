package handlers

import (
	"context"
	"go-auth-boilerplate/internal/database"
	"go-auth-boilerplate/internal/middleware"
	"go-auth-boilerplate/internal/models"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	db       *gorm.DB
	redisURL string
)

func InitHandlers(database *gorm.DB, redis string) {
	db = database
	redisURL = redis
}

var validate = validator.New()

// SignUp godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.User true "User registration info"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /user/signup [post]
func SignUp(c *fiber.Ctx) error {
	var user models.User

	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	result := db.Create(&user)
	if result.Error != nil {
		// Check for duplicate email error
		if strings.Contains(result.Error.Error(), "uni_users_email") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create user",
		})
	}

	// Generate JWT token
	token, err := middleware.CreateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token": token,
	})
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /user/login [post]
func Login(c *fiber.Ctx) error {
	var loginData struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := c.BodyParser(&loginData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(loginData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var user models.User
	if err := db.Where("email = ?", loginData.Email).First(&user).Error; err != nil {
		log.Printf("User not found: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	log.Printf("Found user: ID=%d, Email=%s, Password=%s", user.ID, user.Email, user.Password)
	log.Printf("Attempting to compare password: %s", loginData.Password)

	if err := user.ComparePassword(loginData.Password); err != nil {
		log.Printf("Password comparison failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	token, err := middleware.CreateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create session",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"token": token,
	})
}

// Logout godoc
// @Summary Logout user
// @Description Clear user session
// @Tags auth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Router /user/logout [post]
func Logout(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		ctx := context.Background()
		if err := database.RedisClient.Del(ctx, token).Err(); err != nil {
			log.Printf("Error deleting session from Redis: %v", err)
		}
	}

	cookieToken := c.Cookies("session")
	if cookieToken != "" {
		ctx := context.Background()
		database.RedisClient.Del(ctx, cookieToken)
	}

	c.ClearCookie("session")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Successfully logged out",
	})
}

// UpdatePassword godoc
// @Summary Update user password
// @Description Update the authenticated user's password
// @Tags user
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param passwords body models.PasswordUpdateRequest true "Password update data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /user/update_password [patch]
func UpdatePassword(c *fiber.Ctx) error {
	var passwordData struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=6"`
	}

	if err := c.BodyParser(&passwordData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(passwordData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	userId := c.Locals("user_id").(float64)
	var user models.User
	if err := db.First(&user, uint(userId)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if err := user.ComparePassword(passwordData.CurrentPassword); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid current password",
		})
	}

	user.Password = passwordData.NewPassword
	if err := db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update password",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

// GetSession godoc
// @Summary Get current user session
// @Description Get information about the currently authenticated user
// @Tags user
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /session [get]
func GetSession(c *fiber.Ctx) error {
	userId := c.Locals("user_id").(float64)
	var user models.User
	if err := db.First(&user, uint(userId)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	userResponse := models.UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Age:       user.Age,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return c.Status(fiber.StatusOK).JSON(userResponse)
}

// DeleteUser godoc
// @Summary Delete user account
// @Description Delete the authenticated user's account
// @Tags user
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /user [delete]
func DeleteUser(c *fiber.Ctx) error {
	userId := c.Locals("user_id").(float64)

	token := c.Get("Authorization")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		ctx := context.Background()
		if err := database.RedisClient.Del(ctx, token).Err(); err != nil {
			log.Printf("Error deleting session from Redis: %v", err)
		}
	}

	if err := db.Delete(&models.User{}, uint(userId)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// UpdateUser godoc
// @Summary Update user details
// @Description Update the authenticated user's details
// @Tags user
// @Accept json
// @Produce json
// @Param user body models.User true "User details to update"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /user [patch]
func UpdateUser(c *fiber.Ctx) error {
	var updates models.User
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(float64)
	var user models.User
	if err := db.First(&user, uint(userID)).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if updates.FirstName != "" {
		if len(updates.FirstName) < 2 || len(updates.FirstName) > 50 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "First name must be between 2 and 50 characters",
			})
		}
		user.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		if len(updates.LastName) < 2 || len(updates.LastName) > 50 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Last name must be between 2 and 50 characters",
			})
		}
		user.LastName = updates.LastName
	}
	if updates.Age != 0 {
		if updates.Age < 0 || updates.Age > 150 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Age must be between 0 and 150",
			})
		}
		user.Age = updates.Age
	}

	if err := db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
		"user": fiber.Map{
			"id":         user.ID,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email,
			"age":        user.Age,
		},
	})
}
