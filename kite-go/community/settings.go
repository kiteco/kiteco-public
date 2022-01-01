package community

import (
	"net/mail"
	"os"

	"github.com/kiteco/kiteco/kite-golib/email"
)

// SettingsProvider is an interface that provides settings that are
// shared by the community and enterprise packages, but come
// from different sources
type SettingsProvider interface {
	// Load is called after creating a SettingsProvider and hydrates the settings initial state
	Load() error

	// GetEmailer provides a function that when given a message, will
	// send an email message
	GetEmailer() (func(sender mail.Address, msg email.Message) error, error)

	// GetSenderAddress provides an email address from which
	// to send transactional emails to users
	GetSenderAddress() mail.Address
}

// settings is for enterprise deployment variables
// that are also needed by the public app
type settings struct {
	EmailHostPort string
	EmailUserName string
	EmailPassword string

	EmailSenderName    string
	EmailSenderAddress string
}

// SettingsManager implements SettingsProvider
type SettingsManager struct {
	settings settings
}

// NewSettingsManager creates an new settings manager
func NewSettingsManager() *SettingsManager {
	return &SettingsManager{}
}

// Load the settings for the first time. Run right after NewSettingsManager
func (s *SettingsManager) Load() error {
	s.settings = settings{
		EmailHostPort:      os.Getenv("KITE_ENG_EMAIL_SMTP_HOSTPORT"),
		EmailUserName:      os.Getenv("KITE_ENG_EMAIL_USERNAME"),
		EmailPassword:      os.Getenv("KITE_ENG_EMAIL_PASSWORD"),
		EmailSenderName:    os.Getenv("KITE_ENG_EMAIL_SENDER_NAME"),
		EmailSenderAddress: os.Getenv("KITE_ENG_EMAIL_SENDER_ADDRESS"),
	}
	return nil
}

// GetEmailer returns an emailer function
func (s *SettingsManager) GetEmailer() (func(sender mail.Address, msg email.Message) error, error) {
	client, err := email.NewClient(
		s.settings.EmailHostPort,
		s.settings.EmailUserName,
		s.settings.EmailPassword,
	)
	if err != nil {
		return nil, err
	}

	return func(sender mail.Address, msg email.Message) error {
		return client.Send(sender, msg)
	}, nil
}

// GetSenderAddress returns an email address for transactional emails (sending password-reset, account creation notifications)
func (s *SettingsManager) GetSenderAddress() mail.Address {
	return mail.Address{
		Name:    s.settings.EmailSenderName,
		Address: s.settings.EmailSenderAddress,
	}
}
