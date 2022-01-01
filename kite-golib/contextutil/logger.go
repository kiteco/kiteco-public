package contextutil

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	logPrefix     = "[region=%s release=%s] "
	userLogPrefix = "[region=%s release=%s uid=%d user_ip=%s] "
	logFlags      = log.LstdFlags | log.Lshortfile | log.Lmicroseconds
)

// BasicLogger creates a logger that prefixes logs with the region.
func BasicLogger() *log.Logger {
	// NOTE: if this is changed, the crash log sending in the deployments provisioning script needs
	// to be updated as well, as it depends on looking for the log prefixes
	region := os.Getenv("REGION")
	release := os.Getenv("RELEASE")
	return log.New(os.Stdout, fmt.Sprintf(logPrefix, region, release), logFlags)
}

// UserLoggerFromRequest creates a user logger from the input request
// and prefixes logs with the region, user id, and user ip.
func UserLoggerFromRequest(uid int64, r *http.Request) *log.Logger {
	var (
		region string
		ip     string
	)

	region = os.Getenv("REGION")
	release := os.Getenv("RELEASE")
	ip = r.Header.Get("X-Forwarded-For")
	if ipAddr := net.ParseIP(ip); ipAddr == nil {
		ip = r.RemoteAddr
	}

	return log.New(os.Stdout, fmt.Sprintf(userLogPrefix, region, release, uid, ip), logFlags)
}

// UserLoggerFromParams creates a user logger from the input arguments
// and prefixes logs with the region, user id, and user ip.
func UserLoggerFromParams(uid int64, ip string) *log.Logger {
	region := os.Getenv("REGION")
	release := os.Getenv("RELEASE")
	return log.New(os.Stdout, fmt.Sprintf(userLogPrefix, region, release, uid, ip), logFlags)
}
