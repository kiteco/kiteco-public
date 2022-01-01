// +build standalone

package sidebar

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

func newController(settings component.SettingsManager) devDarwinController {
	return devDarwinController{}
}

type devDarwinController struct{}

// Start implements Controller
func (d devDarwinController) Start() error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	app := filepath.Join(gopath, "src/github.com/kiteco/kiteco/sidebar/dist/mac/Kite.app")
	cmd := exec.Command("open", app)
	return cmd.Run()
}

// Stop implements Controller
func (d devDarwinController) Stop() error {
	return nil
}

// Running implements Controller
func (d devDarwinController) Running() (bool, error) {
	return true, nil
}

// Focus shows the sidebar window if it is hidden and brings it to the front.
func (d devDarwinController) Focus() error {
	return nil
}

// SetWasVisible implements Controller
func (d devDarwinController) SetWasVisible(val bool) error {
	return nil
}

// WasVisible implements Controller
func (d devDarwinController) WasVisible() (bool, error) {
	return true, nil
}

func (d devDarwinController) Notify(id string) error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	app := filepath.Join(gopath, "src/github.com/kiteco/kiteco/sidebar/dist/mac/Kite.app/Contents/MacOS/Kite")
	cmd := exec.Command(app, "--notification="+id)
	return cmd.Run()
}
