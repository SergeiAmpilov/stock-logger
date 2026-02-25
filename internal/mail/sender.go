// mail/sender.go
package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/jordan-wright/email"
)

// EmailConfig holds the configuration for email sending
type EmailConfig struct {
	SMTPServer string
	SMTPPort   int
	Username   string
	Password   string
	Recipients []string
}

// SendReportEmail sends the Excel report as an attachment via email
func SendReportEmail(config EmailConfig, filePath string) error {
	e := email.NewEmail()
	e.From = config.Username
	e.To = config.Recipients
	e.Subject = "Ozon prices and stocks"

	// Attach the Excel file
	_, err := e.AttachFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to attach file: %w", err)
	}

	// Send the email
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPServer)
	err = e.Send(fmt.Sprintf("%s:%d", config.SMTPServer, config.SMTPPort), auth)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully to: %v", config.Recipients)
	return nil
}

// ParseRecipients parses a comma-separated string of email addresses
func ParseRecipients(recipientsStr string) []string {
	recipients := strings.Split(recipientsStr, ",")
	for i, recipient := range recipients {
		recipients[i] = strings.TrimSpace(recipient)
	}
	return recipients
}