package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"stock-logger/internal/config"
	filesrepo "stock-logger/internal/filesxls/repository"
	"stock-logger/internal/mail"
	"stock-logger/internal/reports/repository"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Service handles Excel file operations
type Service struct {
	repo         *repository.DBRepository
	filesXLSRepo *filesrepo.DBRepository
}

// NewService creates a new Excel files service
func NewService(repo *repository.DBRepository, filesXLSRepo *filesrepo.DBRepository) *Service {
	return &Service{
		repo:         repo,
		filesXLSRepo: filesXLSRepo,
	}
}

// GenerateHourlyExcelReport generates an Excel report with data from the last hour
func (s *Service) GenerateHourlyExcelReport() (string, error) {
	// Calculate the date 24 hours ago
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	// Get reports for the last 24 hours
	reports, err := s.repo.GetReportsSince(twentyFourHoursAgo)
	if err != nil {
		return "", err
	}

	// Group reports by article
	articleMap := make(map[string][]repository.Report)
	for _, report := range reports {
		articleMap[report.Article] = append(articleMap[report.Article], report)
	}

	// Extract unique dates and sort them in descending order
	dateSet := make(map[string]bool)
	for _, reportList := range articleMap {
		for _, report := range reportList {
			// Use only the date part for grouping
			datePart := strings.Split(report.RetrievedDate, "T")[0]
			dateSet[datePart] = true
		}
	}

	// Convert set to slice and sort in descending order
	dates := make([]string, 0, len(dateSet))
	for date := range dateSet {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j] // Descending order
	})

	// Create the reports directory if it doesn't exist
	reportsDir := "./file-reports"
	if err := os.MkdirAll(reportsDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %v", err)
	}

	// Generate filename with current timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s.xlsx", timestamp)
	filepath := filepath.Join(reportsDir, filename)

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a sheet for the report
	sheetName := "Report"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// Build headers: Article column + date columns with stock and price sub-columns
	headers := []string{"Article"}
	for _, date := range dates {
		headers = append(headers, fmt.Sprintf("Stock (%s)", date))
		headers = append(headers, fmt.Sprintf("Price (%s)", date))
	}

	// Write headers
	for i, header := range headers {
		cellName := getCellName(i, 0)
		f.SetCellValue(sheetName, cellName, header)
	}

	// Write data rows
	rowIndex := 1 // Start after headers
	for article, reportList := range articleMap {
		// Create a map of date -> (stock, price) for this article
		dateDataMap := make(map[string][2]interface{}) // [0] = stock, [1] = price
		for _, report := range reportList {
			datePart := strings.Split(report.RetrievedDate, "T")[0] // Extract date part
			stock := report.Stock
			var price interface{}
			if report.OurPrice != nil {
				price = *report.OurPrice
			} else {
				price = ""
			}
			dateDataMap[datePart] = [2]interface{}{stock, price}
		}

		// Fill the row for this article
		colIndex := 0
		f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), article)
		colIndex++

		// For each date, fill stock and price
		for _, date := range dates {
			data, exists := dateDataMap[date]
			if exists {
				// Add stock value
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), data[0])
				colIndex++
				// Add price value
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), data[1])
				colIndex++
			} else {
				// Add empty values if no data for this date
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), "")
				colIndex++
				f.SetCellValue(sheetName, getCellName(colIndex, rowIndex), "")
				colIndex++
			}
		}

		rowIndex++
	}

	// Auto-adjust column widths
	for colIdx := 0; colIdx < len(headers); colIdx++ {
		colName := getCellName(colIdx, 0)[:1] // Get just the letter part
		f.SetColWidth(sheetName, colName, colName, 20)
	}

	// Save the Excel file
	err = f.SaveAs(filepath)
	if err != nil {
		return "", err
	}

	// Save file record to database - store only the filename
	err = s.filesXLSRepo.SaveFileRecord(filename)
	if err != nil {
		log.Printf("Warning: Failed to save file record to database: %v", err)
		// We don't return an error here because the file was successfully created
	}

	return filepath, nil
}

// GetAllExcelFiles returns all Excel files from the database
func (s *Service) GetAllExcelFiles() ([]filesrepo.FileXLS, error) {
	return s.filesXLSRepo.GetAllFileRecords()
}

// GenerateAndSendHourlyReport generates an Excel report and sends it via email
func (s *Service) GenerateAndSendHourlyReport(appConfig *config.Config) error {
	log.Println("Generating hourly Excel report...")
	filePath, err := s.GenerateHourlyExcelReport()
	if err != nil {
		log.Printf("Error generating hourly Excel report: %v", err)
		return err
	} else {
		log.Printf("Hourly Excel report generated successfully at: %s", filePath)
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
		err = mail.SendReportEmail(emailConfig, filePath)
		if err != nil {
			log.Printf("Error sending email: %v", err)
			return err
		} else {
			log.Println("Email sent successfully")
		}
	} else {
		log.Println("Email configuration incomplete, skipping email sending")
		log.Printf("SMTP Server: %s, Username: %s, Recipients: %v",
			emailConfig.SMTPServer, emailConfig.Username, emailConfig.Recipients)
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
