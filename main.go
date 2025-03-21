package main

import (
	"fmt"
	"log"

	"go-auth-boilerplate/config"
	"go-auth-boilerplate/docs"
	"go-auth-boilerplate/internal/database"
	"go-auth-boilerplate/internal/middleware"
	"go-auth-boilerplate/internal/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

// @title Go Auth Boilerplate
// @version 1.0
// @description A RESTful API for managing users and their posts
// @host localhost:9999
// @BasePath /api/v1
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	database.InitDB()
	database.InitRedis()

	docs.SwaggerInfo.Title = "Go Auth Boilerplate"
	docs.SwaggerInfo.Description = "A RESTful API for managing users and their posts"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:" + cfg.Server.Port
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(middleware.Logger())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PATCH, DELETE",
	}))

	app.Get("/swagger/*", swagger.HandlerDefault)

	redisURL := fmt.Sprintf("redis://%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	routes.SetupRoutes(app, database.DB, redisURL)

	log.Printf("Server starting on port %s", cfg.Server.Port)
	log.Fatal(app.Listen(":" + cfg.Server.Port))
}
