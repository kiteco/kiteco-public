package kitelog

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	region  = os.Getenv("REGION")
	release = os.Getenv("RELEASE")

	prefix = fmt.Sprintf("[region=%s release=%s] ", region, release)
	flags  = log.LstdFlags | log.Lshortfile | log.Lmicroseconds
)

func init() {
	// for clients still using the standard log package
	log.SetPrefix(prefix)
	log.SetFlags(flags)
}

// Basic prefixes the log line with the region & release identifiers
var Basic = &Logger{
	Default: log.New(os.Stderr, prefix, flags),
}

// NewForUser creates a user logger from the input request
// and prefixes logs with the region, user id, and user ip.
func NewForUser(uid int64, r *http.Request) *Logger {
	ip := r.Header.Get("X-Forwarded-For")
	if ipAddr := net.ParseIP(ip); ipAddr == nil {
		ip = r.RemoteAddr
	}

	prefix = fmt.Sprintf("[region=%s release=%s uid=%d user_ip=%s] ", region, release, uid, ip)

	return &Logger{
		Default: log.New(os.Stderr, prefix, flags),
	}
}

// Logger encapsulates multiple logging handlers
type Logger struct {
	Default   *log.Logger
	Durations Durations
}

// Interface encapsulates the relevant methods of log.Logger
type Interface interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// Printf implements Interface
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Default.Output(2, fmt.Sprintf(format, v...))
}

// Println implements Interface
func (l *Logger) Println(v ...interface{}) {
	l.Default.Output(2, fmt.Sprintln(v...))
}
