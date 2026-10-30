package config

import (
	"log"
	"os"
	"strconv"
	"strings"
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
		Password:      os.Getenv("PASSWORD"),
		JwtSecret:     os.Getenv("JWT_SECRET"),
	}

	// Split email recipients by comma
	recipients := os.Getenv("EMAIL_RECIPIENTS")
	if recipients != "" {
		// Implement splitting logic here if needed
		// For now, just assign the raw string value
		config.EmailRecipients = []string{recipients}
	}

	initTelegram(config)

	return config, nil
}

// initTelegram reads and validates the Telegram delivery settings.
// Invalid or missing settings do not stop the application: Telegram is simply disabled.
func initTelegram(config *Config) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chatID := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))

	if token == "" {
		log.Println("telegram disabled: TELEGRAM_BOT_TOKEN is not set")
		return
	}

	if !isGroupChatID(chatID) {
		log.Printf("telegram disabled: TELEGRAM_CHAT_ID must be a negative group id, got %q", chatID)
		return
	}

	config.TelegramBotToken = token
	config.TelegramChatID = chatID
}

// isGroupChatID reports whether the value is a negative numeric chat id (group or supergroup)
func isGroupChatID(chatID string) bool {
	id, err := strconv.ParseInt(chatID, 10, 64)
	return err == nil && id < 0
}
