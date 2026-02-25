// internal/config/service.go
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPortStr := os.Getenv("SMTP_PORT")
	emailUsername := os.Getenv("EMAIL_USERNAME")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	emailRecipientsStr := os.Getenv("EMAIL_RECIPIENTS")

	if clientID == "" || apiToken == "" {
		return nil, fmt.Errorf("missing required environment variables: CLIENT_ID or API_TOKEN")
	}

	smtpPort := 587 // default port
	if smtpPortStr != "" {
		if port, err := strconv.Atoi(smtpPortStr); err == nil {
			smtpPort = port
		}
	}

	emailRecipients := []string{}
	if emailRecipientsStr != "" {
		emailRecipients = strings.Split(emailRecipientsStr, ",")
		for i, recipient := range emailRecipients {
			emailRecipients[i] = strings.TrimSpace(recipient)
		}
	}

	cs.config = &Config{
		ClientID:      clientID,
		ApiToken:      apiToken,
		SMTPServer:    smtpServer,
		SMTPPort:      smtpPort,
		EmailUsername: emailUsername,
		EmailPassword: emailPassword,
		EmailRecipients: emailRecipients,
	}

	return cs.config, nil
}