package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

// Client is a basic email client
type Client struct {
	Host     string
	Port     string
	User     string
	Password string
}

// Message wraps the recipient list, subject and body
type Message struct {
	HTML    bool
	To      []string
	Bcc     string
	Subject string
	Body    []byte
	Headers map[string]string
}

// NewClient creates a new client
func NewClient(hostport, user, password string) (Client, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return Client{}, err
	}

	if host == "" {
		return Client{}, fmt.Errorf("host not provided")
	}

	if port == "" {
		return Client{}, fmt.Errorf("port not provided")
	}

	if user == "" {
		return Client{}, fmt.Errorf("user not provided")
	}

	if password == "" {
		return Client{}, fmt.Errorf("password not provided")
	}

	return Client{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}, nil
}

// Send sends an email according to the provided Message from the provided
// sender address.
func (s Client) Send(sender mail.Address, m Message) error {
	hostport := net.JoinHostPort(s.Host, s.Port)
	tlsConfig := &tls.Config{ServerName: s.Host}

	client, err := smtp.Dial(hostport)
	if err != nil {
		return err
	}
	defer client.Quit()

	// start TLS
	if err := client.StartTLS(tlsConfig); err != nil {
		return err
	}

	err = client.Auth(smtp.PlainAuth("", s.User, s.Password, s.Host))
	if err != nil {
		return err
	}

	err = client.Mail(sender.Address)
	if err != nil {
		return err
	}

	for _, rcpt := range m.To {
		err = client.Rcpt(rcpt)
		if err != nil {
			return err
		}
	}

	if len(m.Bcc) > 0 {
		err = client.Rcpt(m.Bcc)
		if err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	headers := m.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["From"] = sender.String()
	headers["To"] = strings.Join(m.To, ", ")
	headers["Subject"] = m.Subject

	if m.HTML {
		headers["MIME-version"] = "1.0;"
		headers["Content-Type"] = "text/html; charset=\"UTF-8\";"
	}

	var message bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&message, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(&message, "\r\n")
	message.Write(m.Body)

	_, err = io.Copy(w, &message)
	if err != nil {
		return err
	}

	return nil
}
