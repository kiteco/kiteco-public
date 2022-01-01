package clienttelemetry

import (
	"log"
	"sync"
	"time"

	"github.com/customerio/go-customerio"
	"github.com/dukex/mixpanel"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
)

var (
	// mutex and guarded values
	m             sync.RWMutex
	clientVersion string
	userIDs       userids.IDs
	disabled      bool

	// immutable values
	startTime       = time.Now()
	kiteTracker     = telemetry.NewCommonClient(telemetry.StreamClientEvents)
	mixpanelTracker = mixpanel.New("XXXXXXX", "")
	cioTracker      = customerio.NewCustomerIO("XXXXXXX", "XXXXXXX")
)

// Enable enables tracking
func Enable() {
	m.Lock()
	defer m.Unlock()
	disabled = false
}

// EnableDev enables telemetry to the development CIO/MP
func EnableDev() {
	m.Lock()
	defer m.Unlock()

	mixpanelTracker = mixpanel.New("XXXXXXX", "")
	cioTracker = customerio.NewCustomerIO("XXXXXXX", "XXXXXXX")
	kiteTracker = nil
}

// Disable disables tracking
func Disable() {
	m.Lock()
	defer m.Unlock()
	disabled = true
}

// SetCustomTelemetryClient sets a custom client or nil to disable tracking completely. This will
// set the client for only the kite telemetry source, and nil out the mixpanel source (since both
// KiteEvent and MixpanelEvent send to to segment source, and we want to avoid duplicates
// in test cases, where this method is primarily used)
func SetCustomTelemetryClient(c telemetry.Client) {
	m.Lock()
	defer m.Unlock()

	kiteTracker = c
	mixpanelTracker = nil
	cioTracker = nil
}

// SetClientVersion sets the client version to be used for all subsequent events
func SetClientVersion(newVersion string) {
	m.Lock()
	defer m.Unlock()

	clientVersion = newVersion
}

// SetUserIDs sets the userID and machineID to be used for all subsequent events
func SetUserIDs(ids userids.IDs) {
	log.Printf("track: setting user ids %v", ids)
	m.Lock()
	defer m.Unlock()

	userIDs = ids
}

// GetUserIDs returns the user ID and machine ID that were previously set
func GetUserIDs() userids.IDs {
	m.RLock()
	defer m.RUnlock()

	return userIDs
}

// Close closes the underlying client and resets user and machine ids
func Close() {
	if err := kiteTracker.Close(); err != nil {
		log.Printf("error closing segment tracker: %s", err.Error())
	}
}
