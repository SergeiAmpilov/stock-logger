package service

import (
	"fmt"
	"stock-logger/internal/reports/repository"
	"time"

	"github.com/xuri/excelize/v2"
)

const EXCEL_FILE_PATH = "./report.xlsx"

// Service handles Excel file operations
type Service struct {
	repo *repository.DBRepository
}

// NewService creates a new Excel files service
func NewService(repo *repository.DBRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// GenerateHourlyExcelReport generates an Excel report with data from the last hour
func (s *Service) GenerateHourlyExcelReport() error {
	// Calculate the date one hour ago
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	// Get reports for the last hour
	reports, err := s.repo.GetReportsSince(oneHourAgo)
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
