package main

import (
	"fmt"
	"log"
	"stock-logger/internal/config"
	"stock-logger/internal/ozon"
	"time"
)

const (
	OZON_API_URL     = "https://api-seller.ozon.ru"
	RESTART_INTERVAL = 5 * time.Minute
	DefaultPageSize = 100
)

func main() {
	configService := config.New()
	config, err := configService.Init()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Configuration loaded: ClientID=%s, ApiToken=%s\n", config.ClientID, config.ApiToken)

	ozonSP := ozon.New("https://api-seller.ozon.ru", config.ApiToken, config.ClientID)

	runGetStocks(ozonSP)

	ticker := time.NewTicker(RESTART_INTERVAL)
	defer ticker.Stop()

	for range ticker.C {
		runGetStocks(ozonSP)
	}
}

func runGetStocks(ozonSP *ozon.Service) {
	log.Println("Fetching stock data...")
	response := ozonSP.GetStocks(DefaultPageSize)
	if response != nil {
		log.Printf("Successfully fetched stock data. Total items: %d", response.Total)
	} else {
		log.Println("Failed to fetch stock data")
	}
}
