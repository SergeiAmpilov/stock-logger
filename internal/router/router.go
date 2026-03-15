package router

import (
	"stock-logger/internal/reports/handler"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App, reportsHandler *handler.Handler) {
	// Add routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Stock Logger API is running!")
	})

	app.Get("/api/stocks", reportsHandler.GetReports)
}