package handler

import (
	"log"
	"stock-logger/internal/reports/service"

	"github.com/gofiber/fiber/v2"
)

// QueryParams holds the pagination parameters
type QueryParams struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

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

	// Parse query parameters for pagination
	params := new(QueryParams)
	if err := c.QueryParser(params); err != nil {
		log.Printf("Error parsing query parameters: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Ensure values are not negative
	if params.Limit < 0 {
		params.Limit = 0
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	reports, err := h.service.GetReportsWithPagination(params.Limit, params.Offset)
	if err != nil {
		log.Printf("Error fetching reports: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch stock data",
		})
	}

	// Transform the reports into grouped data by article
	// groupedData := model.ToDataWithPagination(reports, params.Limit, params.Offset)

	return c.JSON(reports)
}
