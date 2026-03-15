package model

import (
	"stock-logger/internal/reports/repository"
)

// ArticleData represents the data for a single article with its history
type ArticleData struct {
	Article string                 `json:"article"`
	Data    []repository.Report `json:"data"`
}

// GetReportsResponse represents the response for getting reports grouped by article
type GetReportsResponse struct {
	Articles []ArticleData `json:"articles"`
}

// ToData transforms a slice of repository reports into grouped article data
func ToData(reports []repository.Report) GetReportsResponse {
	// Group reports by article
	articleMap := make(map[string][]repository.Report)
	
	for _, report := range reports {
		articleMap[report.Article] = append(articleMap[report.Article], report)
	}
	
	// Convert map to slice of ArticleData
	articles := make([]ArticleData, 0, len(articleMap))
	for article, data := range articleMap {
		articles = append(articles, ArticleData{
			Article: article,
			Data:    data,
		})
	}
	
	return GetReportsResponse{
		Articles: articles,
	}
}