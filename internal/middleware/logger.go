package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger middleware logs request details and response status
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Start timer
		start := time.Now()

		// Read request body
		var requestBody []byte
		if c.Request().Body() != nil {
			requestBody = c.Request().Body()
		}

		// Process request
		err := c.Next()
		if err != nil {
			return err
		}

		// Calculate duration
		duration := time.Since(start)

		// Log request and response
		log.Printf("\n=== Request ===\nMethod: %s\nPath: %s\nHeaders: %v\nBody: %s\n\n=== Response ===\nStatus: %d\nDuration: %v\n================\n",
			c.Method(),
			c.Path(),
			c.GetReqHeaders(),
			string(requestBody),
			c.Response().StatusCode(),
			duration,
		)

		return nil
	}
}
