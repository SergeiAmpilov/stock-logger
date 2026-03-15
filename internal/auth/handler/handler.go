package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"stock-logger/internal/auth/model"
	"stock-logger/internal/auth/service"
)

// Handler manages HTTP requests for authentication
type Handler struct {
	service *service.Service
}

// NewHandler creates a new auth handler
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Auth handles the authentication request
func (h *Handler) Auth(c *fiber.Ctx) error {
	log.Println("Handling authentication request")

	req := new(model.AuthRequest)
	if err := c.BodyParser(req); err != nil {
		log.Printf("Error parsing auth request: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	token, err := h.service.Authenticate(req.Password)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid password",
		})
	}

	return c.JSON(model.AuthResponse{
		Token: token,
	})
}