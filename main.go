package main

import (
	"database/sql"
	"fmt"
	"log"
	"stock-logger/internal/config"
	"stock-logger/internal/mail"
	"stock-logger/internal/ozon"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

const (
	OZON_API_URL       = "https://api-seller.ozon.ru"
	RESTART_INTERVAL   = 15 * time.Minute
	DEFAULT_PAGE_SIZE  = 100
	DB_PATH            = "./stocks.db"
	EXCEL_FILE_PATH    = "./report.xlsx"
	HOURLY_REPORT_INTERVAL = 8 * time.Hour  // Every 12 hours
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

	// Open database connection
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Create report table
	err = createReportTable(db)
	if err != nil {
		log.Fatal("Failed to create report table:", err)
	}

	// Run initial stock fetching and saving
	runGetStocksAndSave(db, ozonSP, config)

	// Ticker for API polling every 5 minutes
	apiTicker := time.NewTicker(RESTART_INTERVAL)
	defer apiTicker.Stop()

	// Timer for hourly report generation (every 12 hours)
	hourlyReportTicker := time.NewTicker(HOURLY_REPORT_INTERVAL)
	defer hourlyReportTicker.Stop()

	for {
		select {
		case <-apiTicker.C:
			runGetStocksAndSave(db, ozonSP, config)
		case <-hourlyReportTicker.C:
			// Generate and send hourly report
			runGenerateAndSendHourlyReport(db, config)
		}
	}
}

func runGetStocksAndSave(db *sql.DB, ozonSP *ozon.Service, appConfig *config.Config) {
	log.Println("Fetching stock data...")
	stockResponse := ozonSP.GetStocks(DEFAULT_PAGE_SIZE)
	if stockResponse != nil {
		log.Printf("Successfully fetched stock data. Total items: %d", stockResponse.Total)
	} else {
		log.Println("Failed to fetch stock data")
		return
	}

	log.Println("Fetching price data...")
	priceResponse := ozonSP.GetAllPrices(DEFAULT_PAGE_SIZE)
	if priceResponse != nil {
		log.Printf("Successfully fetched price data. Total items: %d", priceResponse.Total)
	} else {
		log.Println("Failed to fetch price data")
		return
	}

	// Combine stock and price data and save to report table
	now := time.Now()
	err := saveCombinedReport(db, stockResponse, priceResponse, now)
	if err != nil {
		log.Printf("Error saving combined report to database: %v", err)
	} else {
		log.Println("Combined report saved to database successfully")
	}
}

// Function to handle hourly report generation and email sending
func runGenerateAndSendHourlyReport(db *sql.DB, appConfig *config.Config) {
	log.Println("Generating hourly Excel report...")
	err := generateHourlyExcelReport(db)
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

func createReportTable(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		article TEXT,
		stock INTEGER,
		our_price REAL
	);
	`
	_, err := db.Exec(sqlStmt)
	return err
}

func saveCombinedReport(db *sql.DB, stockResponse *ozon.GetStockDataResponse, priceResponse *ozon.GetPriceDataResponse, reportTime time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO reports(retrieved_date, article, stock, our_price) VALUES(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	reportTimeStr := reportTime.Format(time.RFC3339)

	// Create a map of prices by OfferID for quick lookup
	priceMap := make(map[string]ozon.Price)
	for _, item := range priceResponse.Items {
		priceMap[item.OfferID] = item.Price
	}

	// Process stock data and match with prices
	for _, item := range stockResponse.Items {
		for _, stock := range item.Stocks {
			// Only process stocks with type "fbs"
			if stock.Type == "fbs" {
				priceInfo, exists := priceMap[item.OfferID]
				if exists {
					_, err = stmt.Exec(
						reportTimeStr,
						item.OfferID,    // Article
						stock.Present,   // Stock
						priceInfo.Price, // Our Price
					)
					if err != nil {
						return err
					}
				} else {
					// If no price info exists for this item, insert with NULL prices
					_, err = stmt.Exec(
						reportTimeStr,
						item.OfferID,  // Article
						stock.Present, // Stock
						nil,           // Our Price
					)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return tx.Commit()
}

// Generate Excel report with data from the last hour
func generateHourlyExcelReport(db *sql.DB) error {
	// Calculate the date one hour ago
	oneHourAgo := time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	
	// Query reports for the last hour ordered by date descending (newest first)
	rows, err := db.Query(`
		SELECT retrieved_date, article, stock, our_price 
		FROM reports 
		WHERE retrieved_date >= ?
		ORDER BY retrieved_date DESC
	`, oneHourAgo)
	if err != nil {
		return err
	}
	defer rows.Close()

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
	rowIndex := 1 // Start after headers
	for rows.Next() {
		var retrievedDate, article string
		var stock int
		var ourPrice *float64

		err := rows.Scan(&retrievedDate, &article, &stock, &ourPrice)
		if err != nil {
			return err
		}

		// Write the row data
		f.SetCellValue(sheetName, getCellName(0, rowIndex), retrievedDate)
		f.SetCellValue(sheetName, getCellName(1, rowIndex), article)
		f.SetCellValue(sheetName, getCellName(2, rowIndex), stock)
		
		if ourPrice != nil {
			f.SetCellValue(sheetName, getCellName(3, rowIndex), *ourPrice)
		} else {
			f.SetCellValue(sheetName, getCellName(3, rowIndex), "")
		}
		
		rowIndex++
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