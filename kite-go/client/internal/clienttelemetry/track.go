package clienttelemetry

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/dukex/mixpanel"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
)

// IDType is the type of ID to use for telemetry
type IDType uint8

// IDTypes
const (
	None IDType = iota
	MetricsID
	ForgetfulMetricsID
	InstallID
	UserID
)

func (i IDType) getRLocked() string {
	switch i {
	case MetricsID:
		return userIDs.MetricsID()
	case ForgetfulMetricsID:
		return userIDs.ForgetfulMetricsID()
	case InstallID:
		return userIDs.InstallID()
	case UserID:
		return fmt.Sprintf("%d", userIDs.UserID())
	default:
		panic("no ID")
	}
}

// Options specifies identifiers for each destination
type Options struct {
	mp   IDType
	cio  IDType
	kite IDType
}

// Empty is the no-op Options
var Empty = Options{}

// Default is the default Options that most clients should use
var Default = Options{
	mp:   MetricsID,
	cio:  ForgetfulMetricsID,
	kite: None,
}

// MP sets the Mixpanel IDType
func (o Options) MP(t IDType) Options {
	o.mp = t
	return o
}

// CIO sets the Customer.io IDType
func (o Options) CIO(t IDType) Options {
	o.cio = t
	return o
}

// Kite sets the Kite IDType
func (o Options) Kite(t IDType) Options {
	o.kite = t
	return o
}

// CIOOnly disables Mixpanel and Kite
func (o Options) CIOOnly() Options {
	o.kite = None
	o.mp = None
	return o
}

// MPOnly disables Customer.io and Kite
func (o Options) MPOnly() Options {
	o.kite = None
	o.cio = None
	return o
}

// Event sends an event
func (o Options) Event(name string, props map[string]interface{}) {
	m.RLock()
	defer m.RUnlock()

	mp := mixpanelTracker != nil && o.mp != None
	cio := cioTracker != nil && o.cio != None
	kite := kiteTracker != nil && o.kite != None
	if !mp && !cio && !kite {
		return
	}

	eventProps, ok := eventDataLocked(name, props)
	if !ok {
		return
	}

	if mp {
		err := mixpanelTracker.Track(o.mp.getRLocked(), name, &mixpanel.Event{
			Properties: eventProps,
		})
		if err != nil {
			log.Printf("error sending Mixpanel event %s: %v", name, err)
		}
	}

	if cio {
		err := cioTracker.Track(o.cio.getRLocked(), name, eventProps)
		if err != nil {
			log.Printf("error sending Customer.io event %s: %v", name, err)
		}
	}

	if kite {
		err := kiteTracker.Track(context.Background(), o.kite.getRLocked(), name, eventProps)
		if err != nil {
			log.Printf("error sending Kite event %s: %v", name, err)
		}
	}
}

// Update updates properties
func (o Options) Update(props map[string]interface{}) {
	m.RLock()
	defer m.RUnlock()

	mp := mixpanelTracker != nil && o.mp != None
	cio := cioTracker != nil && o.cio != None

	if disabled {
		log.Printf("tracker.Update: disabled")
		return
	}

	if mp {
		err := mixpanelTracker.Update(o.mp.getRLocked(), &mixpanel.Update{
			Operation:  "$set",
			Properties: props,
		})
		if err != nil {
			log.Println("error updating mixpanel user properties", err)
		}
	}

	if cio {
		err := cioTracker.Identify(o.cio.getRLocked(), props)
		if err != nil {
			log.Println("error updating customer.io user properties", err)
		}
	}

	// Kite doesn't track properties
}

// eventDataLocked returns the full set of properties to send. The boolean return value if the
// properties are valid. If the tracking is disabled or if userId isn't set, then false is returned.
func eventDataLocked(name string, props map[string]interface{}) (map[string]interface{}, bool) {
	log.Printf("tracker.Event: %s", name)
	if disabled {
		log.Printf("tracker.Event: disabled")
		return nil, false
	}

	if userIDs == nil {
		log.Println("missing userids")
		return nil, false
	}

	properties := make(map[string]interface{})
	properties["user_id"] = userIDs.UserID()
	properties["install_id"] = userIDs.InstallID()
	properties["machine"] = userIDs.MachineID()
	properties["platform"] = runtime.GOOS
	properties["client_version"] = clientVersion
	properties["client_uptime_ns"] = time.Since(startTime)
	properties["kite_local"] = true
	for k, v := range props {
		properties[k] = v
	}
	telemetry.AugmentProps(properties)

	return properties, true
}
