package service

import (
	"log"
	"stock-logger/internal/ozon"
	"stock-logger/internal/reports/repository"
	"time"
)

const DEFAULT_PAGE_SIZE = 100

// Service handles business logic for reports
type Service struct {
	repo   *repository.DBRepository
	ozonSP *ozon.Service
}

// NewService creates a new reports service
func NewService(repo *repository.DBRepository, ozon *ozon.Service) *Service {
	return &Service{
		repo:   repo,
		ozonSP: ozon,
	}
}

// GetAllReports retrieves all reports from the database
func (s *Service) GetAllReports() ([]repository.Report, error) {
	// Pass an empty time value to get all reports
	return s.repo.GetReportsSince(time.Time{})
}

// RunGetStocksAndSave fetches stock and price data and saves to the database
func (s *Service) RunGetStocksAndSave() {
	log.Println("Fetching stock data...")
	stockResponse := s.ozonSP.GetStocks(DEFAULT_PAGE_SIZE)
	if stockResponse != nil {
		log.Printf("Successfully fetched stock data. Total items: %d", stockResponse.Total)
	} else {
		log.Println("Failed to fetch stock data")
		return
	}

	log.Println("Fetching price data...")
	priceResponse := s.ozonSP.GetAllPrices(DEFAULT_PAGE_SIZE)
	if priceResponse != nil {
		log.Printf("Successfully fetched price data. Total items: %d", priceResponse.Total)
	} else {
		log.Println("Failed to fetch price data")
		return
	}

	// Combine stock and price data and save to report table
	now := time.Now()
	err := s.repo.SaveCombinedReport(stockResponse, priceResponse, now)
	if err != nil {
		log.Printf("Error saving combined report to database: %v", err)
	} else {
		log.Println("Combined report saved to database successfully")
	}
}
