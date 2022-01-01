package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/email"
)

var (
	addr = mail.Address{
		Name:    "Kite Engineering",
		Address: "eng@kite.com",
	}
	hostport = os.Getenv("KITE_ENG_EMAIL_SMTP_HOSTPORT")
	user     = os.Getenv("KITE_ENG_EMAIL_USERNAME")
	password = os.Getenv("KITE_ENG_EMAIL_PASSWORD")
)

func main() {
	var (
		to      string
		subject string
		body    string
		html    bool
	)

	flag.StringVar(&to, "to", "", "comma separated list of recipients of the email")
	flag.StringVar(&subject, "subject", "", "subject of the email")
	flag.StringVar(&body, "body", "", "body of the email - can be a filename or a string")
	flag.BoolVar(&html, "html", false, "set true to send HTML-formatted email")
	flag.Parse()

	client, err := email.NewClient(hostport, user, password)
	if err != nil {
		log.Fatalln("unable to start email client:", err)
	}

	var recipients []string
	for _, r := range strings.Split(to, ",") {
		recipients = append(recipients, strings.TrimSpace(r))
	}

	var r io.Reader
	if _, err := os.Stat(body); os.IsNotExist(err) {
		r = strings.NewReader(body)
	} else {
		f, err := os.Open(body)
		if err != nil {
			log.Fatalln("unable to open file:", err)
		}
		r = f
	}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalln("unable to read message:", err)
	}

	message := email.Message{
		To:      recipients,
		Subject: subject,
		Body:    buf,
		HTML:    html,
	}

	err = client.Send(addr, message)
	if err != nil {
		log.Fatalln(err)
	}
}
