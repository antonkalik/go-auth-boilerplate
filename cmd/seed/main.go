package main

import (
	"fmt"
	"go-auth-boilerplate/internal/database"
	"go-auth-boilerplate/seeds"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func main() {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	envFile := filepath.Join("config", fmt.Sprintf("%s.env", env))

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: %s file not found", envFile)
	}

	database.InitDB()

	if err := seeds.Seed(); err != nil {
		log.Fatalf("Error seeding database: %v", err)
	}

	log.Println("Database seeded successfully!")
}
