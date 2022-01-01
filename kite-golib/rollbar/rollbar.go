package rollbar

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/userids"
	rollbar "github.com/rollbar/rollbar-go"
)

var (
	withPanic   = false
	logDisabled = false
	userID      userids.IDs
	// accept every 3rd message on average, at most one every 500ms
	accepted = newRollbarLimiter(3, 500*time.Millisecond)
)

// SetClientVersion sets the client version for segment tracking; it should be called early on in the lifecycle
func SetClientVersion(ver string) {
	rollbar.SetCodeVersion(ver)
}

func init() {
	// This environment variable will be set on the deployed machines.
	// Without this token, reporting is a NOOP, which is default behavior
	// (e.g while debugging/developing in a VM).
	rollbar.SetToken(os.Getenv("ROLLBAR_TOKEN"))

	// Default to "development" environment unless on public facing
	// production machines. Those machines will have this environment
	// variable set accordingly.
	env := os.Getenv("ROLLBAR_ENV")
	if env == "" {
		env = "development"
	}
	rollbar.SetEnvironment(env)

	// Set initial empty userids
	SetUserIDs(&userids.UserIDs{})
}

// Disable rollbar messages
func Disable() {
	rollbar.SetToken("")
	rollbar.SetEnvironment("")
	rollbar.SetEnabled(false)
}

// WithPanic causes all subsequent rollbar calls to panic. The returned function reverts the behavior.
// Intended for use as: defer rollbar.WithPanic(t)() within a test function.
// Note that this should be called within the main goroutine, and isn't thread-safe.
func WithPanic(testing.TB) func() {
	withPanic = true
	return func() {
		withPanic = false
	}
}

// DebugSetPanic manually set the panic mode. Very unsafe and should only be used for debugging Kite.
func DebugSetPanic() {
	withPanic = true
}

// SetLogDisabled sets the status of logging to Golang's log.
func SetLogDisabled(disabled bool) {
	logDisabled = disabled
}

// SetToken sets the token
func SetToken(token string) {
	rollbar.SetToken(token)
}

// SetEnvironment sets the environment
func SetEnvironment(env string) {
	rollbar.SetEnvironment(env)
}

// SetUserIDs sets the userids object so rollbar can report userid/installid
func SetUserIDs(uids userids.IDs) {
	userID = uids
}

// Wait will block until the queue of errors / messages is empty.
func Wait() {
	rollbar.Wait()
}

// Critical sends a critical error report to Rollbar.
func Critical(err error, data ...interface{}) {
	send(rollbar.CRIT, err, data...)
}

// Error sends an error report to Rollbar.
func Error(err error, data ...interface{}) {
	send(rollbar.ERR, err, data...)
}

// Warning sends a warning report to Rollbar.
func Warning(err error, data ...interface{}) {
	send(rollbar.WARN, err, data...)
}

// Info sends a warning report to Rollbar.
func Info(err error, data ...interface{}) {
	send(rollbar.INFO, err, data...)
}

// Debug sends a debug report to Rollbar.
func Debug(err error, data ...interface{}) {
	send(rollbar.DEBUG, err, data...)
}

// --

// PanicRecovery send a panic report to rollbar
func PanicRecovery(r interface{}, data ...interface{}) {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, false)
	logPrintf("panic: %s\n%s", r, buf[:n])
	err := fmt.Errorf("panic: %s", r)
	send(rollbar.ERR, err, data...)
}

// RequestError sends an error report to Rollbar with the given request-specific
// information.
func RequestError(err error, r *http.Request) {
	skip := 2 // Go up two stack frames to report where the error came from
	rollbar.RequestErrorWithStackSkip(rollbar.ERR, r, err, skip)
}

// RecoveryError sends an error report to Rollbar with the given request-specific
// information. This is a variant of RequestError() that should be called from
// within recovery functions.
func RecoveryError(err error, r *http.Request) {
	// Need to skip more stack frames for recovery...
	skip := 4 // Go up four stack frames to report where the error came from
	rollbar.RequestErrorWithStackSkip(rollbar.ERR, r, err, skip)
}

// --

func send(level string, err error, data ...interface{}) {
	logPrintln("ROLLBAR", level, err, data)
	if withPanic {
		panic(fmt.Sprintf("rollbar [%s]: %v %v", level, err, data))
	}
	if rollbar.Token() == "" {
		// If there is no token, we are most likely in a dev environment, so log the error to help with debugging
		// TODO: switch more tests/binaries over to using WithPanic
		logPrintf("rollbar [%s]: %v %v", level, err, data)
		return
	}

	if !accepted() {
		logPrintln("dropping rollbar event due to filtering")
		return
	}

	// Set all the IDs we have, because they could all be useful depending on what we're trying to debug
	extras := map[string]interface{}{
		"user_id":    userID.UserID(),
		"install_id": userID.InstallID(),
		"machine_id": userID.MachineID(),
		"metrics_id": userID.MetricsID(),
	}
	for idx, d := range data {
		extras[fmt.Sprintf("data%d", idx)] = d
	}
	skip := 2 // Go up two stack frames to report where the error came from
	rollbar.ErrorWithStackSkipWithExtras(level, err, skip, extras)
}

func logPrintf(format string, v ...interface{}) {
	if !logDisabled {
		log.Printf(format, v...)
	}
}

func logPrintln(v ...interface{}) {
	if !logDisabled {
		log.Println(v...)
	}
}
