package main

import (
	"database/sql"
	"fmt"
	"log"
	"stock-logger/internal/config"
	"stock-logger/internal/ozon"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	OZON_API_URL     = "https://api-seller.ozon.ru"
	RESTART_INTERVAL = 5 * time.Minute
	DefaultPageSize  = 100
	DB_PATH          = "./stocks.db"
)

func main() {
	configService := config.New()
	config, err := configService.Init()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Configuration loaded: ClientID=%s, ApiToken=%s\n", config.ClientID, config.ApiToken)

	ozonSP := ozon.New("https://api-seller.ozon.ru", config.ApiToken, config.ClientID)

	// Open database connection
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Create table if not exists
	err = createStockTable(db)
	if err != nil {
		log.Fatal("Failed to create table:", err)
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
	response := ozonSP.GetStocks(10)
	if response != nil {
		log.Printf("Successfully fetched stock data. Total items: %d", response.Total)
		err := saveStocksToDB(db, response)
		if err != nil {
			log.Printf("Error saving stocks to database: %v", err)
		} else {
			log.Println("Stocks saved to database successfully")
		}
	} else {
		log.Println("Failed to fetch stock data")
	}
}

func createStockTable(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS stocks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		sku TEXT,
		type TEXT,
		stock INTEGER
	);
	`
	_, err := db.Exec(sqlStmt)
	return err
}

func saveStocksToDB(db *sql.DB, response *ozon.GetStockDataResponse) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO stocks(retrieved_date, sku, type, stock) VALUES(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)

	for _, item := range response.Items {
		for _, stock := range item.Stocks {
			_, err = stmt.Exec(now, item.OfferID, stock.Type, stock.Present)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
