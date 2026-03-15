package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"stock-logger/internal/filesxls/service"
)

// Handler manages HTTP requests for Excel files
type Handler struct {
	service *service.Service
}

// NewHandler creates a new Excel files handler
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GenerateReport handles the request to generate an Excel report
func (h *Handler) GenerateReport(c *fiber.Ctx) error {
	log.Println("Handling request to generate Excel report")

	filePath, err := h.service.GenerateHourlyExcelReport()
	if err != nil {
		log.Printf("Error generating Excel report: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate Excel report",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Excel report generated successfully",
		"filePath": filePath,
	})
}

// ListFiles handles the request to list all Excel report files
func (h *Handler) ListFiles(c *fiber.Ctx) error {
	log.Println("Handling request to list Excel report files")

	files, err := h.service.GetAllExcelFiles()
	if err != nil {
		log.Printf("Error listing Excel files: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list Excel files",
		})
	}

	return c.JSON(files)
}