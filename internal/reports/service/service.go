package service

import (
	"time"
	"stock-logger/internal/reports/repository"
)

// Service handles business logic for reports
type Service struct {
	repo *repository.DBRepository
}

// NewService creates a new reports service
func NewService(repo *repository.DBRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetAllReports retrieves all reports from the database
func (s *Service) GetAllReports() ([]repository.Report, error) {
	// Pass an empty time value to get all reports
	return s.repo.GetReportsSince(time.Time{})
}