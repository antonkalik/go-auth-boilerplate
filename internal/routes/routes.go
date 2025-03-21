package routes

import (
	"go-auth-boilerplate/internal/handlers"
	"go-auth-boilerplate/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, redisURL string) {
	handlers.InitHandlers(db, redisURL)

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api.Post("/user/signup", handlers.SignUp)
	api.Post("/user/login", handlers.Login)
	api.Post("/user/logout", handlers.Logout)

	protected := api.Use(middleware.Protected())

	protected.Get("/session", handlers.GetSession)
	protected.Patch("/user", handlers.UpdateUser)
	protected.Patch("/user/update_password", handlers.UpdatePassword)
	protected.Delete("/user", handlers.DeleteUser)

	protected.Post("/posts/create", handlers.CreatePost)
	protected.Get("/posts", handlers.GetPosts)
	protected.Get("/posts/:id", handlers.GetPost)
	protected.Patch("/posts/:id/update", handlers.UpdatePost)
	protected.Delete("/posts/:id/delete", handlers.DeletePost)
}
