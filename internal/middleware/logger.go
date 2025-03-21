package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		var requestBody []byte
		if c.Request().Body() != nil {
			requestBody = c.Request().Body()
		}

		err := c.Next()
		if err != nil {
			return err
		}

		duration := time.Since(start)

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
