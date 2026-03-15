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
}
