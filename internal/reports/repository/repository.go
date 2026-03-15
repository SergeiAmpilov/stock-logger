package repository

import (
	"database/sql"
	"time"

	"stock-logger/internal/ozon"
	_ "github.com/mattn/go-sqlite3"
)

// Report represents a single report entry from the database
type Report struct {
	RetrievedDate string
	Article       string
	Stock         int
	OurPrice      *float64
}

// DBRepository handles all database operations for reports
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository creates a new database repository
func NewDBRepository(dbPath string) (*DBRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	
	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	
	repo := &DBRepository{db: db}
	
	// Create tables if they don't exist
	if err := repo.CreateReportTable(); err != nil {
		db.Close()
		return nil, err
	}
	
	return repo, nil
}

// Close closes the database connection
func (r *DBRepository) Close() error {
	return r.db.Close()
}

// CreateReportTable creates the reports table if it doesn't exist
func (r *DBRepository) CreateReportTable() error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		retrieved_date DATETIME,
		article TEXT,
		stock INTEGER,
		our_price REAL
	);
	`
	_, err := r.db.Exec(sqlStmt)
	return err
}

// SaveCombinedReport saves combined stock and price data to the database
func (r *DBRepository) SaveCombinedReport(stockResponse *ozon.GetStockDataResponse, priceResponse *ozon.GetPriceDataResponse, reportTime time.Time) error {
	tx, err := r.db.Begin()
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

// GetReportsSince retrieves reports from a specific time
func (r *DBRepository) GetReportsSince(fromTime time.Time) ([]Report, error) {
	fromTimeStr := fromTime.Format("2006-01-02 15:04:05")
	
	rows, err := r.db.Query(`
		SELECT retrieved_date, article, stock, our_price 
		FROM reports 
		WHERE retrieved_date >= ?
		ORDER BY retrieved_date DESC
	`, fromTimeStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		err := rows.Scan(&report.RetrievedDate, &report.Article, &report.Stock, &report.OurPrice)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return reports, nil
}

// GetAllReports retrieves all reports from the database without filtering by time
func (r *DBRepository) GetAllReports() ([]Report, error) {
	rows, err := r.db.Query(`
		SELECT retrieved_date, article, stock, our_price 
		FROM reports 
		ORDER BY retrieved_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		err := rows.Scan(&report.RetrievedDate, &report.Article, &report.Stock, &report.OurPrice)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return reports, nil
}
