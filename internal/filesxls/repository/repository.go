package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileXLS represents a single Excel file record from the database
type FileXLS struct {
	ID          int
	CreatedAt   string
	FilePath    string
}

// DBRepository handles all database operations for Excel files
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository creates a new database repository for Excel files
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
	if err := repo.CreateFilesXLSTable(); err != nil {
		db.Close()
		return nil, err
	}
	
	return repo, nil
}

// Close closes the database connection
func (r *DBRepository) Close() error {
	return r.db.Close()
}

// CreateFilesXLSTable creates the filesxls table if it doesn't exist
func (r *DBRepository) CreateFilesXLSTable() error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS filesxls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		file_path TEXT
	);
	`
	_, err := r.db.Exec(sqlStmt)
	return err
}

// SaveFileRecord saves a new Excel file record to the database
func (r *DBRepository) SaveFileRecord(filePath string) error {
	query := `INSERT INTO filesxls(created_at, file_path) VALUES(?, ?)`
	createdAt := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(query, createdAt, filePath)
	if err != nil {
		return fmt.Errorf("failed to save file record: %v", err)
	}
	return nil
}

// GetAllFileRecords retrieves all Excel file records from the database
func (r *DBRepository) GetAllFileRecords() ([]FileXLS, error) {
	rows, err := r.db.Query(`
		SELECT id, created_at, file_path
		FROM filesxls
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileXLS
	for rows.Next() {
		var file FileXLS
		err := rows.Scan(&file.ID, &file.CreatedAt, &file.FilePath)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return files, nil
}