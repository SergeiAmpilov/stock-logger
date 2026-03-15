package model

import (
	"math"
	"stock-logger/internal/reports/repository"
)

// ArticleData represents the data for a single article with its history
type ArticleData struct {
	Article string                 `json:"article"`
	Data    []repository.Report `json:"data"`
}

// ToDataWithPagination transforms a slice of repository reports into grouped article data with pagination
func ToDataWithPagination(reports []repository.Report, limit, offset int) GetReportsResponse {
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
	
	// Calculate total number of articles before pagination
	totalArticles := len(articles)
	
	// Apply pagination to articles
	// Calculate actual start index
	startIndex := offset
	if startIndex > totalArticles {
		startIndex = totalArticles
	}
	
	// Calculate end index based on limit
	endIndex := totalArticles
	if limit > 0 {
		endIndex = startIndex + limit
		if endIndex > totalArticles {
			endIndex = totalArticles
		}
	}
	
	// Slice articles according to pagination
	if startIndex < totalArticles {
		articles = articles[startIndex:endIndex]
	} else {
		articles = []ArticleData{}
	}
	
	// Calculate total pages
	totalPages := 0
	if limit > 0 && totalArticles > 0 {
		totalPages = int(math.Ceil(float64(totalArticles) / float64(limit)))
	}
	
	return GetReportsResponse{
		Articles: articles,
		PageInfo: PageInfo{
			Limit:       limit,
			Offset:      offset,
			Total:       totalArticles,
			TotalPages:  totalPages,
		},
	}
}

// PageInfo contains pagination information
type PageInfo struct {
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// GetReportsResponse represents the response for getting reports grouped by article
type GetReportsResponse struct {
	Articles []ArticleData `json:"articles"`
	PageInfo PageInfo      `json:"page_info,omitempty"`
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