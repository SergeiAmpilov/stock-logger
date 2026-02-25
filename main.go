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
	RESTART_INTERVAL = 1 * time.Hour
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

	ozonSP := ozon.New(OZON_API_URL, config.ApiToken, config.ClientID)

	// Open database connection
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Create tables if not exist
	err = createStockTable(db)
	if err != nil {
		log.Fatal("Failed to create stock table:", err)
	}

	err = createPriceTable(db)
	if err != nil {
		log.Fatal("Failed to create price table:", err)
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
	response := ozonSP.GetStocks(DefaultPageSize)
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

	log.Println("Fetching price data...")
	priceResponse := ozonSP.GetAllPrices(DefaultPageSize) // Changed to GetPrices instead of GetAllPrices
	if priceResponse != nil {
		log.Printf("Successfully fetched price data. Total items: %d", priceResponse.Total)
		err := savePricesToDB(db, priceResponse)
		if err != nil {
			log.Printf("Error saving prices to database: %v", err)
		} else {
			log.Println("Prices saved to database successfully")
		}
	} else {
		log.Println("Failed to fetch price data")
	}
}

func createStockTable(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS stocks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		sku TEXT,
		stock INTEGER
	);
	`
	_, err := db.Exec(sqlStmt)
	return err
}

func createPriceTable(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS prices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		sku TEXT,
		price REAL,
		old_price REAL,
		min_price REAL,
		marketing_seller_price REAL,
		retail_price REAL,
		currency_code TEXT
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

	stmt, err := tx.Prepare("INSERT INTO stocks(retrieved_date, sku, stock) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)

	for _, item := range response.Items {
		for _, stock := range item.Stocks {
			// Only process stocks with type "fbs"
			if stock.Type == "fbs" {
				_, err = stmt.Exec(now, item.OfferID, stock.Present)
				if err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func savePricesToDB(db *sql.DB, response *ozon.GetPriceDataResponse) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO prices(retrieved_date, sku, price, old_price, min_price, marketing_seller_price, retail_price, currency_code) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)

	for _, item := range response.Items {
		_, err = stmt.Exec(
			now, 
			item.OfferID, 
			item.Price.Price,
			item.Price.OldPrice,
			item.Price.MinPrice,
			item.Price.MarketingSellerPrice,
			item.Price.RetailPrice,
			item.Price.CurrencyCode,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}