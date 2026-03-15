package service

import (
	"log"
	"stock-logger/internal/ozon"
	"stock-logger/internal/reports/model"
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
func NewService(repo *repository.DBRepository, ozonSP *ozon.Service) *Service {
	return &Service{
		repo:   repo,
		ozonSP: ozonSP,
	}
}

// GetAllReports retrieves all reports from the database
func (s *Service) GetAllReports() ([]repository.Report, error) {
	// Pass an empty time value to get all reports
	return s.repo.GetReportsSince(time.Time{})
}

// GetReportsWithPagination retrieves reports with pagination applied at the article level
func (s *Service) GetReportsWithPagination(limit, offset int) (*model.GetReportsResponse, error) {

	resp := &model.GetReportsResponse{
		Articles: make([]model.ArticleData, 0),
		PageInfo: model.PageInfo{},
	}
	// Pass an empty time value to get all reports
	allReports, err := s.repo.GetAllReports()
	if err != nil {
		return nil, err
	}

	// Group reports by article first
	articleMap := make(map[string][]repository.Report)
	for _, report := range allReports {
		articleMap[report.Article] = append(articleMap[report.Article], report)
	}

	// Convert map keys to slice to apply pagination
	articles := make([]string, 0, len(articleMap))
	for article := range articleMap {
		articles = append(articles, article)
	}

	// Apply offset and limit to articles
	if offset >= len(articles) {
		return resp, nil
	}

	startIndex := offset
	endIndex := len(articles)

	if limit > 0 {
		endIndex = startIndex + limit
		if endIndex > len(articles) {
			endIndex = len(articles)
		}
	}

	selectedArticles := articles[startIndex:endIndex]

	for _, article := range selectedArticles {
		resp.Articles = append(resp.Articles, model.ArticleData{
			Article: article,
			Data:    articleMap[article],
		})
	}

	if limit > 0 {
		resp.PageInfo = model.PageInfo{
			Limit:      limit,
			Offset:     offset,
			Total:      len(articles),
			TotalPages: (len(articles) + limit - 1) / limit,
		}
	}

	return resp, nil

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
