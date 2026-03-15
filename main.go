package main

import (
	"fmt"
	"log"
	"stock-logger/internal/config"
	handler_files "stock-logger/internal/filesxls/handler"
	filesrepo "stock-logger/internal/filesxls/repository"
	service_files "stock-logger/internal/filesxls/service"

	"stock-logger/internal/ozon"

	reports_handler "stock-logger/internal/reports/handler"
	"stock-logger/internal/reports/repository"
	reports_service "stock-logger/internal/reports/service"
	"stock-logger/internal/router"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

const (
	OZON_API_URL           = "https://api-seller.ozon.ru"
	RESTART_INTERVAL       = 15 * time.Minute
	DEFAULT_PAGE_SIZE      = 100
	DB_PATH                = "./stocks.db"
	HOURLY_REPORT_INTERVAL = 8 * time.Hour // Every 12 hours
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	configService := config.New()
	config, err := configService.Init()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Configuration loaded: ClientID=%s, ApiToken=%s\n", config.ClientID, config.ApiToken)

	ozonSP := ozon.New(OZON_API_URL, config.ApiToken, config.ClientID)

	// Initialize database repository
	repo, err := repository.NewDBRepository(DB_PATH)
	if err != nil {
		log.Fatal("Failed to initialize database repository:", err)
	}
	defer repo.Close()

	// Initialize filesxls database repository
	filesXLSRepo, err := filesrepo.NewDBRepository(DB_PATH)
	if err != nil {
		log.Fatal("Failed to initialize filesxls database repository:", err)
	}
	defer filesXLSRepo.Close()

	// Initialize reports service and handler
	reportsService := reports_service.NewService(repo, ozonSP)
	reportsHandler := reports_handler.NewHandler(reportsService)

	// Initialize Excel files service and handler
	excelService := service_files.NewService(repo, filesXLSRepo)
	excelHandler := handler_files.NewHandler(excelService)

	// Initialize HTML template engine
	engine := html.New("./views", ".html") // Using views directory for templates

	// Initialize Fiber app with template engine
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Use(logger.New())

	// Setup routes using the router package
	router.SetupRoutes(app, reportsHandler, excelHandler)

	// Start the Fiber server in a separate goroutine
	go func() {
		port := config.Port
		if port == "" {
			port = "3000" // default port
		}
		log.Printf("Starting Fiber server on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Fiber server failed to start: %v", err)
		}
	}()

	// Run initial stock fetching and saving
	reportsService.RunGetStocksAndSave()

	// Ticker for API polling every 5 minutes
	apiTicker := time.NewTicker(RESTART_INTERVAL)
	defer apiTicker.Stop()

	// Timer for hourly report generation (every 12 hours)
	hourlyReportTicker := time.NewTicker(HOURLY_REPORT_INTERVAL)
	defer hourlyReportTicker.Stop()

	for {
		select {
		case <-apiTicker.C:
			reportsService.RunGetStocksAndSave()
		case <-hourlyReportTicker.C:
			excelService.GenerateAndSendHourlyReport(config)
		}
	}
}
