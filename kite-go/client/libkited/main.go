package main

/*
#cgo darwin CFLAGS: -mmacosx-version-min=10.11
#cgo darwin LDFLAGS: -mmacosx-version-min=10.11
*/
import "C"
import (
	"fmt"
	"log"
	"os"
	"sync"

	clientpkg "github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates/liveupdates"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/client/visibility"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/throttle"
)

var (
	mu     sync.Mutex
	client *clientpkg.Client
)

//export kiteInitialize
func kiteInitialize() (ret bool) {
	defer panicRecoveryBool(&ret)

	if err := throttle.SetLowPriority(); err != nil {
		log.Printf("failed to set low process priority: %s", err)
	}

	mu.Lock()
	defer mu.Unlock()

	var err error
	target, err := liveupdates.UpdateTarget()
	if err != nil {
		clientapp.Alert(err)
		rollbar.Critical(fmt.Errorf("kiteInitialize: error initializing libkited: %v", err))
		log.Println("kiteInitialize: error initializing libkited:", err)
		return false
	}

	client, err = clientapp.Start(&clientpkg.Options{Updater: liveupdates.NewManager(target)})
	if err != nil {
		clientapp.Alert(err)
		rollbar.Critical(fmt.Errorf("kiteInitialize: error initializing libkited: %v", err))
		log.Println("kiteInitialize: error initializing libkited:", err)
		return false
	}

	return true
}

//export kiteConnect
func kiteConnect() (ret bool) {
	defer panicRecoveryBool(&ret)

	mu.Lock()
	defer mu.Unlock()
	if client == nil {
		log.Println("kiteConnectToHost: run KiteInitialize before connecting")
		return false
	}

	go func() {
		err := client.Connect(client.Settings.Server())
		if err != nil {
			log.Printf("libkited failed to connect: %v", err)
			// TODO: have the outer function wait for confirmation that kite is connected
			// before returning, and then on failure have it return false.
		}
	}()

	return true
}

//export kiteSetEnv
func kiteSetEnv(key, value *C.char) {
	defer panicRecovery()

	keyStr := C.GoString(key)
	valueStr := C.GoString(value)
	os.Setenv(keyStr, valueStr)
}

//export kiteTrackSidebarVisibility
// kiteTrackSidebarVisibility stores state about whether the sidebar was visible at the
// time the method is invoked
func kiteTrackSidebarVisibility() {
	defer panicRecovery()

	mu.Lock()
	defer mu.Unlock()
	if client == nil {
		log.Println("kiteTrackSidebar: run KiteInitialize before connecting")
		return
	}

	sidebar.SetRestartIfPreviouslyVisible(visibility.RecentlyVisible())
}

//export kiteStopSidebar
// kiteStopSidebar stops the sidebar application if it is running
func kiteStopSidebar() {
	defer panicRecovery()
	sidebar.Stop()
}

//export kiteCheckForUpdates
// kiteCheckForUpdates is called when the user explicitly checks for updates
func kiteCheckForUpdates(showModal bool) {
	defer panicRecovery()

	mu.Lock()
	defer mu.Unlock()
	if client == nil {
		log.Println("kiteCheckForUpdates: run KiteInitialize before connecting")
		return
	}

	client.Updater.CheckForUpdates(showModal)
}

//export kiteUpdateReady
// kiteUpdateReady returns true if an update has been downloaded and is waiting to
// be installed when the process terminates.
func kiteUpdateReady() bool {
	defer panicRecovery()

	mu.Lock()
	defer mu.Unlock()
	if client == nil {
		log.Println("kiteUpdateReady: run KiteInitialize before connecting")
		return false
	}

	return client.Updater.UpdateReady()
}

func panicRecoveryBool(ret *bool) {
	if err := recover(); err != nil {
		rollbar.PanicRecovery(err)
		*ret = false
	}
}

func panicRecovery() {
	if err := recover(); err != nil {
		rollbar.PanicRecovery(err)
	}
}

func main() {
	// Required by CGO. Does nothing. See exported API above
}
