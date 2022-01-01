package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/envutil"
)

const (
	instURI      = "collector.instrumentalapp.com:8001" // instrumental collection agent URI
	certPem      = "certs/client.pem"                   // client cert for TLS
	certKey      = "certs/client.key"                   // client cert for TLS
	agentVersion = "instrumental/0.0.3"                 // version string to send in hello
)

var (
	tlsConfig tls.Config // TLS config for connection to instrumental collection agent
	hostname  string     // hostname to send in hello
	instToken string     // authentication token for instrumental
)

func init() {
	// Load client certs
	cert, err := tls.LoadX509KeyPair(certPem, certKey)
	if err != nil {
		log.Fatalf("error loading client certs: %v", err)
	}
	// TLS config from certs
	tlsConfig = tls.Config{Certificates: []tls.Certificate{cert}}

	// Get hostname
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("error getting hostname: %v", err)
	}
	// Get auth token
	instToken = envutil.MustGetenv("INSTRUMENTAL_TOKEN")
}

// send messages to instrumental via a new TLS connection
func sendInstrumental(messages []string) error {
	conn, err := tls.Dial("tcp", instURI, &tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// send hello
	fmt.Fprintf(conn, "hello version golang/%s hostname %s\n", agentVersion, hostname)
	// listen for reply
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if reply != "ok\n" {
		return fmt.Errorf("non-ok reply from server: %s", reply)
	}

	// send auth
	fmt.Fprintf(conn, "authenticate %s\n", instToken)
	// listen for reply
	reply, err = bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if reply != "ok\n" {
		return fmt.Errorf("non-ok reply from server: %s", reply)
	}

	// NOTE: after auth, instrumental will take all further communication to be metrics and will not
	// reply. Invalid metrics will cause the connection to be closed, so we must attempt a read after
	// each send to make sure that we have not lost connection.
	//
	// ...is what the documentation says; in reality, the connection does not close, but it will stop
	// accepting inputs after being sent an invalidly formatted input, so instead we do a regex
	// format check for each message and also keep track of the 120s TCP timeout.

	// TCP timeout timer
	t := time.NewTimer(120 * time.Second)
	defer t.Stop()

	for i, msg := range messages {
		// check timer
		select {
		case <-t.C:
			return fmt.Errorf("connection closed; failed to send %d of %d messages", len(messages)-i, len(messages))
		default:
		}

		// validate
		if !validMessage(msg) {
			log.Printf("'%s' is an invalid input", msg)
			continue
		}
		// send message
		fmt.Fprintf(conn, msg+"\n")
	}

	return nil
}

// validMessage checks that the messge is correctly formatted as a metric message
func validMessage(message string) bool {
	// must be exactly three tokens separated by spaces
	tokens := strings.Split(message, " ")
	if len(tokens) != 3 {
		return false
	}
	// first token must be either gauge or increment
	if tokens[0] != "gauge" && tokens[0] != "increment" {
		return false
	}
	// second token must be alphanumeric/hyphen/underscore separated by periods
	// (regex taken from instrumental python agent)
	match, err := regexp.MatchString(`^([\d\w\-_]+\.)*[\d\w\-_]+$`, tokens[1])
	if err != nil {
		log.Fatalf("regex error: %v", err)
	}
	if !match {
		return false
	}

	// third token must be an integer or float (regex taken from instrumental python agent)
	match, err = regexp.MatchString(`^-?\d+(\.\d+)?(e-\d+)?$`, tokens[2])
	if err != nil {
		log.Fatalf("regex error: %v", err)
	}
	if !match {
		return false
	}

	return true
}

// clean up status names
func instName(str string) string {
	var output string
	for _, i := range str {
		c := string(i)
		// lowercase
		c = strings.ToLower(c)
		// if hyphen, replace with underscore
		if c == "-" {
			c = "_"
		}

		// if space or period, replace with hyphen
		if c == " " || c == "." {
			c = "-"
		}
		// check if valid character, only add if it is
		match, err := regexp.MatchString(`[\d\w\-_\.]+`, c)
		if err != nil {
			log.Fatalf("regex error: %v", err)
		}
		if match {
			output = output + c
		}
	}

	return output
}
