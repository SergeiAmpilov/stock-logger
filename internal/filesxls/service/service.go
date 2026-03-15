package service

import (
	"fmt"
	"log"
	"time"
	"stock-logger/internal/config"
	"stock-logger/internal/mail"
	"stock-logger/internal/reports/repository"

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

// GenerateAndSendHourlyReport generates an Excel report and sends it via email
func (s *Service) GenerateAndSendHourlyReport(appConfig *config.Config) error {
	log.Println("Generating hourly Excel report...")
	err := s.GenerateHourlyExcelReport()
	if err != nil {
		log.Printf("Error generating hourly Excel report: %v", err)
		return err
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