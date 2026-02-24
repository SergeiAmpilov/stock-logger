package ozon

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Service struct {
	BaseURL  string
	ApiToken string
	ClientID string

	GetStoks *ApiEndpoint
}

func New(baseURL, apiToken, clientID string) *Service {
	return &Service{
		BaseURL:  baseURL,
		ApiToken: apiToken,
		ClientID: clientID,
		GetStoks: &ApiEndpoint{
			Method:   "POST",
			Endpoint: "/v4/product/info/stocks",
		},
	}
}

func (s *Service) GetStocks(limit int) *GetStockDataResponse {
	allItems := []Item{}
	var cursor string
	hasMorePages := true

	for hasMorePages {
		request := GetStockDataRequest{
			Cursor: cursor,
			Filter: make(map[string]interface{}),
			Limit:  limit,
		}

		requestBody, err := json.Marshal(request)
		if err != nil {
			log.Printf("Error marshaling request: %v", err)
			return nil
		}

		url := s.BaseURL + s.GetStoks.Endpoint
		req, err := http.NewRequest(s.GetStoks.Method, url, bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return nil
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Client-Id", s.ClientID)
		req.Header.Set("Api-Key", s.ApiToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making request: %v", err)
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Request failed with status: %d", resp.StatusCode)
			return nil
		}

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			return nil
		}

		var response GetStockDataResponse
		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			return nil
		}

		allItems = append(allItems, response.Items...)

		// Check if there are more pages
		cursor = response.Cursor
		hasMorePages = cursor != ""
	}

	// Return aggregated response
	return &GetStockDataResponse{
		Items:  allItems,
		Total:  len(allItems),
		Cursor: "",
	}
}
