package router

import (
	filesxls_handler "stock-logger/internal/filesxls/handler"
	reports_handler "stock-logger/internal/reports/handler"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App, reportsHandler *reports_handler.Handler, excelHandler *filesxls_handler.Handler) {
	// Add routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Stock Logger Dashboard",
		}) // Render HTML template instead of sending plain text
	})

	app.Get("/api/stocks", reportsHandler.GetReports)
	
	// Add route for Excel report generation
	app.Post("/api/excel/generate", excelHandler.GenerateReport)
}