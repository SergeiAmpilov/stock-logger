package main

import (
	"fmt"
	"log"
	"stock-logger/internal/config"
	handler_files "stock-logger/internal/filesxls/handler"
	service_files "stock-logger/internal/filesxls/service"
	"stock-logger/internal/mail"
	"stock-logger/internal/ozon"

	reports_handler "stock-logger/internal/reports/handler"
	"stock-logger/internal/reports/repository"
	reports_service "stock-logger/internal/reports/service"
	"stock-logger/internal/router"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

const (
	OZON_API_URL           = "https://api-seller.ozon.ru"
	RESTART_INTERVAL       = 15 * time.Minute
	DEFAULT_PAGE_SIZE      = 100
	DB_PATH                = "./stocks.db"
	EXCEL_FILE_PATH        = "./report.xlsx"
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

	// Initialize reports service and handler
	reportsService := reports_service.NewService(repo, ozonSP)
	reportsHandler := reports_handler.NewHandler(reportsService)

	// Initialize Excel files service and handler
	excelService := service_files.NewService(repo)
	excelHandler := handler_files.NewHandler(excelService)

	// Initialize Fiber app
	app := fiber.New()
	app.Use(logger.New())

	// Setup routes using the router package
	router.SetupRoutes(app, reportsHandler, excelHandler)

	// Run initial stock fetching and saving
	reportsService.RunGetStocksAndSave(ozonSP)

	// Ticker for API polling every 5 minutes
	apiTicker := time.NewTicker(RESTART_INTERVAL)
	defer apiTicker.Stop()

	// Timer for hourly report generation (every 12 hours)
	hourlyReportTicker := time.NewTicker(HOURLY_REPORT_INTERVAL)
	defer hourlyReportTicker.Stop()

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

	for {
		select {
		case <-apiTicker.C:
			reportsService.RunGetStocksAndSave(ozonSP)
		case <-hourlyReportTicker.C:
			// Generate and send hourly report
			runGenerateAndSendHourlyReport(excelService, config)
		}
	}
}

// Function to handle hourly report generation and email sending
func runGenerateAndSendHourlyReport(excelService *service_files.Service, appConfig *config.Config) {
	log.Println("Generating hourly Excel report...")
	err := excelService.GenerateHourlyExcelReport()
	if err != nil {
		log.Printf("Error generating hourly Excel report: %v", err)
	} else {
		log.Println("Hourly Excel report generated successfully")
	}

	// Send email with the report
	emailConfig := mail.EmailConfig{
		SMTPServer: appConfig.SMTPServer,
		SMTPPort:   appConfig.SMTPPort,
		Username:   appConfig.EmailUsername,
		Password:   appConfig.EmailPassword,
		Recipients: appConfig.EmailRecipients,
	}

	if emailConfig.Username != "" && emailConfig.Password != "" && len(emailConfig.Recipients) > 0 {
		log.Printf("Attempting to send email to: %v", emailConfig.Recipients)
		err = mail.SendReportEmail(emailConfig, EXCEL_FILE_PATH)
		if err != nil {
			log.Printf("Error sending email: %v", err)
		} else {
			log.Println("Email sent successfully")
		}
	} else {
		log.Println("Email configuration incomplete, skipping email sending")
		log.Printf("SMTP Server: %s, Username: %s, Recipients: %v",
			emailConfig.SMTPServer, emailConfig.Username, emailConfig.Recipients)
	}
}
