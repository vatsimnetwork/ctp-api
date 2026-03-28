package middleware

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/vatsimnetwork/ctp-api/config"
)

func APIKeyAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "missing api key",
			})
		}

		req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, config.C.AuthServiceURL+"/internal/apikey/validate", nil)
		if err != nil {
			log.Error().Err(err).Msg("apikey: failed to create validation request")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "an internal error occurred",
			})
		}
		req.Header.Set("X-API-Key", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("apikey: auth service unreachable")
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error":   "bad_gateway",
				"message": "auth service unreachable",
			})
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			c.Set("Content-Type", "application/json")
			return c.Status(resp.StatusCode).Send(body)
		}

		var result struct {
			Valid    bool `json:"valid"`
			ReadOnly bool `json:"readOnly"`
		}
		if err := json.Unmarshal(body, &result); err != nil || !result.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "invalid api key",
			})
		}

		if result.ReadOnly && isWriteMethod(c.Method()) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "this api key is read-only",
			})
		}

		return c.Next()
	}
}

func isWriteMethod(method string) bool {
	return method == fiber.MethodPost || method == fiber.MethodPut || method == fiber.MethodDelete || method == fiber.MethodPatch
}
