package main

import (
	"fmt"
	"log"
	"stock-logger/internal/config"
	"stock-logger/internal/mail"
	"stock-logger/internal/ozon"
	"stock-logger/internal/reports/handler"
	"stock-logger/internal/reports/repository"
	"stock-logger/internal/reports/service"
	"stock-logger/internal/router"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
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
	reportsService := service.NewService(repo, ozonSP)
	reportsHandler := handler.NewHandler(reportsService)

	// Initialize Fiber app
	app := fiber.New()
	app.Use(logger.New())

	// Setup routes using the router package
	router.SetupRoutes(app, reportsHandler)

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
			runGenerateAndSendHourlyReport(repo, config)
		}
	}
}

// Function to handle hourly report generation and email sending
func runGenerateAndSendHourlyReport(repo *repository.DBRepository, appConfig *config.Config) {
	log.Println("Generating hourly Excel report...")
	err := generateHourlyExcelReport(repo)
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

// Generate Excel report with data from the last hour
func generateHourlyExcelReport(repo *repository.DBRepository) error {
	// Calculate the date one hour ago
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	// Get reports for the last hour
	reports, err := repo.GetReportsSince(oneHourAgo)
	if err != nil {
		return err
	}

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a sheet for the report
	sheetName := "Report"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// Define headers
	headers := []string{"Retrieved Date", "Article", "Stock", "Our Price"}

	// Write headers
	for i, header := range headers {
		cellName := getCellName(i, 0)
		f.SetCellValue(sheetName, cellName, header)
	}

	// Write data rows
	for i, report := range reports {
		rowIndex := i + 1 // Start after headers

		f.SetCellValue(sheetName, getCellName(0, rowIndex), report.RetrievedDate)
		f.SetCellValue(sheetName, getCellName(1, rowIndex), report.Article)
		f.SetCellValue(sheetName, getCellName(2, rowIndex), report.Stock)

		if report.OurPrice != nil {
			f.SetCellValue(sheetName, getCellName(3, rowIndex), *report.OurPrice)
		} else {
			f.SetCellValue(sheetName, getCellName(3, rowIndex), "")
		}
	}

	// Auto-adjust column widths
	for col := 'A'; col <= 'D'; col++ {
		f.SetColWidth(sheetName, string(col), string(col), 20)
	}

	// Save the Excel file
	err = f.SaveAs(EXCEL_FILE_PATH)
	if err != nil {
		return err
	}

	return nil
}

// Helper function to convert column index to Excel column name (A, B, ..., Z, AA, AB, ...)
func getCellName(colIndex, rowIndex int) string {
	colName := ""
	colNum := colIndex + 1

	for colNum > 0 {
		colNum--
		colName = string(rune(colNum%26+'A')) + colName
		colNum /= 26
	}

	return fmt.Sprintf("%s%d", colName, rowIndex+1)
}
