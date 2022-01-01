//go:generate goversioninfo -64 -o resource_windows.syso -icon app.ico

package main

//#cgo windows LDFLAGS: -L${SRCDIR}/../../../../windows/tensorflow/lib
//#cgo windows LDFLAGS: -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
//#cgo linux CFLAGS: -fno-builtin-malloc -fno-builtin-calloc -fno-builtin-realloc -fno-builtin-free
//#cgo linux LDFLAGS: -L${SRCDIR}/../../../../linux/tcmalloc -ltcmalloc_minimal -Wl,-rpath,$ORIGIN
import "C"

import (
	"log"
	"os"

	clientpkg "github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates/liveupdates"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/throttle"
)

func fatal(err error) {
	switch err {
	// ErrRunning occurs when kite's port was already in use
	// but kite was manually launched. In this case, we just launch
	// the sidebar. We shouldn't rollbar or alert anything.
	case clientapp.ErrRunning:
	default:
		rollbar.Critical(err)
		clientapp.Alert(err)
	}

	log.Println(err)
	os.Exit(1)
}

func main() {
	// Do not call arg.MustParse or flag.Parse here. Flags are handled by clientapp

	if err := throttle.SetLowPriority(); err != nil {
		log.Printf("failed to set low process priority: %s", err)
	}

	target, err := liveupdates.UpdateTarget()
	if err != nil {
		fatal(err)
	}

	client, err := clientapp.Start(&clientpkg.Options{Updater: liveupdates.NewManager(target)})
	if err == clientapp.ErrAutostartDisabled {
		log.Println(err)
		os.Exit(0)
	}
	if err != nil {
		rollbar.Critical(err)
		log.Println(err)
		fatal(err)
	}

	err = client.Connect(client.Settings.Server())
	if err != nil {
		fatal(err)
	}

	// sleep forever
	select {}
}
