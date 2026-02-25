// internal/config/model.go
package config

type Config struct {
	ClientID        string
	ApiToken        string
	SMTPServer      string
	SMTPPort        int
	EmailUsername   string
	EmailPassword   string
	EmailRecipients []string
}
