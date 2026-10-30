package config

// Config holds the application configuration
type Config struct {
	ClientID        string
	ApiToken        string
	SMTPServer      string
	SMTPPort        int
	EmailUsername   string
	EmailPassword   string
	EmailRecipients []string
	Port            string
	Password        string
	JwtSecret       string

	TelegramBotToken string
	TelegramChatID   string // negative id of the private group, single target for all publications
}
