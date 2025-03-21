package seeds

import (
	"errors"
	"go-auth-boilerplate/internal/database"
	"go-auth-boilerplate/internal/models"
	"log"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

func generateRandomPost(userID uint) models.Post {
	titles := []string{
		"My First Post", "A Great Day", "Thoughts on Technology",
		"Future Plans", "Random Ideas", "Project Update",
		"Learning Experience", "New Discovery", "Interesting Findings",
		"Personal Growth",
	}

	bodies := []string{
		"This is a detailed post about my experiences...",
		"Today I learned something fascinating...",
		"I've been thinking about this project...",
		"Here are my thoughts on recent developments...",
		"Let me share an interesting story...",
	}

	return models.Post{
		Title:  titles[rand.Intn(len(titles))] + " " + time.Now().Format("2006-01-02"),
		Body:   bodies[rand.Intn(len(bodies))] + " " + time.Now().Format("15:04:05"),
		UserID: userID,
	}
}

func createUserIfNotExists(user *models.User) error {
	var existingUser models.User
	err := database.DB.Where("email = ?", user.Email).First(&existingUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := user.BeforeSave(nil); err != nil {
			return err
		}
		if err := database.DB.Create(user).Error; err != nil {
			return err
		}
		log.Printf("Created new user: %s %s", user.FirstName, user.LastName)
		return nil
	} else if err != nil {
		return err
	}

	user.ID = existingUser.ID
	log.Printf("User already exists: %s %s", user.FirstName, user.LastName)
	return nil
}

func Seed() error {
	rand.Seed(time.Now().UnixNano())

	testUser := models.User{
		FirstName: "Anton",
		LastName:  "Kalik",
		Age:       40,
		Email:     "antonkalik@gmail.com",
		Password:  "Pass123",
	}

	if err := createUserIfNotExists(&testUser); err != nil {
		log.Printf("Error handling test user: %v", err)
		return err
	}

	var postCount int64
	database.DB.Model(&models.Post{}).Where("user_id = ?", testUser.ID).Count(&postCount)

	remainingPosts := 20 - int(postCount)
	if remainingPosts > 0 {
		for i := 0; i < remainingPosts; i++ {
			post := generateRandomPost(testUser.ID)
			if err := database.DB.Create(&post).Error; err != nil {
				log.Printf("Error creating post %d: %v", i+1, err)
			}
		}
		log.Printf("Created %d new posts for user %s", remainingPosts, testUser.Email)
	}

	additionalUsers := []models.User{
		{
			FirstName: "John",
			LastName:  "Doe",
			Age:       30,
			Email:     "john@example.com",
			Password:  "Pass123",
		},
		{
			FirstName: "Jane",
			LastName:  "Smith",
			Age:       25,
			Email:     "jane@example.com",
			Password:  "Pass123",
		},
	}

	for _, user := range additionalUsers {
		if err := createUserIfNotExists(&user); err != nil {
			log.Printf("Error handling additional user: %v", err)
		}
	}

	return nil
}
