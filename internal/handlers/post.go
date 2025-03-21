package handlers

import (
	"go-auth-boilerplate/internal/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// CreatePost godoc
// @Summary Create a new post
// @Description Create a new post for the authenticated user
// @Tags posts
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param post body models.Post true "Post creation info"
// @Success 201 {object} models.Post
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /posts/create [post]
func CreatePost(c *fiber.Ctx) error {
	var post models.Post

	if err := c.BodyParser(&post); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(post); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	userId := uint(c.Locals("user_id").(float64))
	post.UserID = userId

	if err := db.Create(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create post",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}

// GetPosts godoc
// @Summary Get user posts
// @Description Get all posts for the authenticated user with pagination
// @Tags posts
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} models.PostsResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /posts [get]
func GetPosts(c *fiber.Ctx) error {
	userId := uint(c.Locals("user_id").(float64))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	var posts []models.Post
	var total int64

	if err := db.Model(&models.Post{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch posts",
		})
	}

	if err := db.Where("user_id = ?", userId).Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch posts",
		})
	}

	hasNext := (offset + len(posts)) < int(total)

	return c.Status(fiber.StatusOK).JSON(models.PostsResponse{
		TotalItems: int(total),
		Items:      posts,
		Limit:      limit,
		HasNext:    hasNext,
	})
}

// GetPost godoc
// @Summary Get a specific post
// @Description Get a specific post by ID for the authenticated user
// @Tags posts
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Post ID"
// @Success 200 {object} models.Post
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /posts/{id} [get]
func GetPost(c *fiber.Ctx) error {
	userId := uint(c.Locals("user_id").(float64))
	postId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid post ID",
		})
	}

	var post models.Post
	if err := db.Where("id = ? AND user_id = ?", postId, userId).First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

// UpdatePost godoc
// @Summary Update a post
// @Description Update a specific post by ID
// @Tags posts
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Post ID"
// @Param post body models.PostUpdateRequest true "Post update info"
// @Success 200 {object} models.Post
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /posts/{id}/update [patch]
func UpdatePost(c *fiber.Ctx) error {
	userId := uint(c.Locals("user_id").(float64))
	postId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid post ID",
		})
	}

	var post models.Post
	if err := db.Where("id = ? AND user_id = ?", postId, userId).First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	var updateData struct {
		Title string `json:"title" validate:"required,min=3,max=100"`
		Body  string `json:"body" validate:"required,min=10"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if updateData.Title != "" {
		post.Title = updateData.Title
	}
	if updateData.Body != "" {
		post.Body = updateData.Body
	}

	if err := db.Save(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update post",
		})
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

// DeletePost godoc
// @Summary Delete a post
// @Description Delete a specific post by ID
// @Tags posts
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Post ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /posts/{id}/delete [delete]
func DeletePost(c *fiber.Ctx) error {
	userId := uint(c.Locals("user_id").(float64))
	postId, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid post ID",
		})
	}

	result := db.Where("id = ? AND user_id = ?", postId, userId).Delete(&models.Post{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete post",
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Post not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Post deleted successfully",
	})
}
