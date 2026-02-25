package main

import (
	"database/sql"
	"fmt"
	"log"
	"stock-logger/internal/config"
	"stock-logger/internal/ozon"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

const (
	OZON_API_URL     = "https://api-seller.ozon.ru"
	RESTART_INTERVAL = 5 * time.Minute
	DefaultPageSize  = 100
	DB_PATH          = "./stocks.db"
	EXCEL_FILE_PATH  = "./report.xlsx"
)

func main() {
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

	runGetStocksAndSave(db, ozonSP)

	ticker := time.NewTicker(RESTART_INTERVAL)
	defer ticker.Stop()

	for range ticker.C {
		runGetStocksAndSave(db, ozonSP)
	}
}

func runGetStocksAndSave(db *sql.DB, ozonSP *ozon.Service) {
	log.Println("Fetching stock data...")
	stockResponse := ozonSP.GetStocks(DefaultPageSize)
	if stockResponse != nil {
		log.Printf("Successfully fetched stock data. Total items: %d", stockResponse.Total)
	} else {
		log.Println("Failed to fetch stock data")
		return
	}

	log.Println("Fetching price data...")
	priceResponse := ozonSP.GetAllPrices(DefaultPageSize)
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

	// Generate Excel report
	err = generateExcelReport(db)
	if err != nil {
		log.Printf("Error generating Excel report: %v", err)
	} else {
		log.Println("Excel report generated successfully")
	}
}

func createReportTable(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		article TEXT,
		stock INTEGER,
		ozon_price REAL,
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

	stmt, err := tx.Prepare("INSERT INTO reports(retrieved_date, article, stock, ozon_price, our_price) VALUES(?, ?, ?, ?, ?)")
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
						item.OfferID,                   // Article
						stock.Present,                  // Stock
						priceInfo.MarketingSellerPrice, // Ozon Price
						priceInfo.Price,                // Our Price
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
						nil,           // Ozon Price
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

func generateExcelReport(db *sql.DB) error {
	// Query all reports ordered by date descending (newest first)
	rows, err := db.Query(`
		SELECT retrieved_date, article, stock, ozon_price, our_price 
		FROM reports 
		ORDER BY retrieved_date DESC
	`)
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
	headers := []string{"Артикул"}
	dateHeaders := make(map[string]bool)
	var dates []string

	// Collect all unique dates and articles
	records := make(map[string]map[string][]interface{})

	for rows.Next() {
		var retrievedDate, article string
		var stock int
		var ozonPrice, ourPrice *float64

		err := rows.Scan(&retrievedDate, &article, &stock, &ozonPrice, &ourPrice)
		if err != nil {
			return err
		}

		// Add date to headers if not already added
		if !dateHeaders[retrievedDate] {
			dateHeaders[retrievedDate] = true
			dates = append(dates, retrievedDate)
		}

		// Initialize map for this article if not exists
		if records[article] == nil {
			records[article] = make(map[string][]interface{})
		}

		// Store data for this date
		record := []interface{}{stock}
		if ozonPrice != nil {
			record = append(record, *ozonPrice)
		} else {
			record = append(record, "")
		}
		if ourPrice != nil {
			record = append(record, *ourPrice)
		} else {
			record = append(record, "")
		}
		records[article][retrievedDate] = record
	}

	// Write headers
	colIndex := 0
	f.SetCellValue(sheetName, getCellName(colIndex, 0), headers[0]) // Article header
	colIndex++

	for _, date := range dates {
		// Date header
		f.SetCellValue(sheetName, getCellName(colIndex, 0), date)
		// Sub-headers for Stock, Ozon Price, Our Price
		f.SetCellValue(sheetName, getCellName(colIndex, 1), "Остаток")
		f.SetCellValue(sheetName, getCellName(colIndex+1, 1), "Цена озон")
		f.SetCellValue(sheetName, getCellName(colIndex+2, 1), "Цена наша")
		colIndex += 3
	}

	// Write data rows
	rowIndex := 2 // Start after headers
	for article, dateData := range records {
		colIndex := 0
		// Write article
		f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), article)
		colIndex++

		// For each date, write the corresponding data
		for _, date := range dates {
			data, exists := dateData[date]
			if exists && len(data) >= 3 {
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), data[0])   // Stock
				f.SetCellValue(sheetName, getCellName(colIndex+1, rowIndex), data[1]) // Ozon Price
				f.SetCellValue(sheetName, getCellName(colIndex+2, rowIndex), data[2]) // Our Price
			} else {
				// Fill with empty values if no data exists for this date
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), "")
				f.SetCellValue(sheetName, getCellName(colIndex+1, rowIndex), "")
				f.SetCellValue(sheetName, getCellName(colIndex+2, rowIndex), "")
			}
			colIndex += 3
		}
		rowIndex++
	}

	// Auto-adjust column widths
	for col := 'A'; col <= rune('A'+len(dates)*3); col++ {
		f.SetColWidth(sheetName, string(col), string(col), 15)
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
