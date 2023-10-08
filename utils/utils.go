package utils

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"os"
)

func GetEnvOrDefault(envKey string, defaultValue string) string {
	env := os.Getenv(envKey)
	if env == "" {
		env = defaultValue
	}
	return env
}

func UnmarshalBody(c *fiber.Ctx, v any) error {
	body := c.Body()
	err := json.Unmarshal(body, v)
	return err
}
