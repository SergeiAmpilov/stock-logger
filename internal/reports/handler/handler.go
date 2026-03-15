package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"stock-logger/internal/reports/model"
	"stock-logger/internal/reports/service"
)

// Handler manages HTTP requests for reports
type Handler struct {
	service *service.Service
}

// NewHandler creates a new reports handler
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetReports handles the request to retrieve all reports
func (h *Handler) GetReports(c *fiber.Ctx) error {
	log.Println("Handling request to fetch all reports")
	
	reports, err := h.service.GetAllReports()
	if err != nil {
		log.Printf("Error fetching reports: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch stock data",
		})
	}
	
	// Transform the reports into grouped data by article
	groupedData := model.ToData(reports)
	
	return c.JSON(groupedData)
}