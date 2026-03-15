package config

import (
	"os"
	"strconv"
)

// Service handles configuration management
type Service struct{}

// New creates a new configuration service
func New() *Service {
	return &Service{}
}

// Init initializes the configuration from environment variables
func (s *Service) Init() (*Config, error) {
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpPort := 587 // default port
	if smtpPortStr != "" {
		if port, err := strconv.Atoi(smtpPortStr); err == nil {
			smtpPort = port
		}
	}

	config := &Config{
		ClientID:      os.Getenv("CLIENT_ID"),
		ApiToken:      os.Getenv("API_TOKEN"),
		SMTPServer:    os.Getenv("SMTP_SERVER"),
		SMTPPort:      smtpPort,
		EmailUsername: os.Getenv("EMAIL_USERNAME"),
		EmailPassword: os.Getenv("EMAIL_PASSWORD"),
		Port:          os.Getenv("PORT"),
	}

	// Split email recipients by comma
	recipients := os.Getenv("EMAIL_RECIPIENTS")
	if recipients != "" {
		// Implement splitting logic here if needed
		// For now, just assign the raw string value
		config.EmailRecipients = []string{recipients}
	}

	return config, nil
}
