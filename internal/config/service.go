package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type ConfigService struct {
	config *Config
}

func New() *ConfigService {
	return &ConfigService{}
}

func (cs *ConfigService) Init() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading: %v", err)
	}

	clientID := os.Getenv("CLIENT_ID")
	apiToken := os.Getenv("API_TOKEN")

	if clientID == "" || apiToken == "" {
		return nil, fmt.Errorf("missing required environment variables: CLIENT_ID or API_TOKEN")
	}

	cs.config = &Config{
		ClientID: clientID,
		ApiToken: apiToken,
	}

	return cs.config, nil
}
