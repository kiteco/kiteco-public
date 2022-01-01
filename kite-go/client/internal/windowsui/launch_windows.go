package windowsui

import (
	"log"
	"os/exec"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/reg"
)

func run(args ...string) error {
	installdir, err := reg.InstallPath()
	if err != nil {
		installdir = `C:\Program Files\Kite`
	}
	exepath := filepath.Join(installdir, "KiteOnboarding.exe")
	log.Println("running onboarding:", exepath)
	return exec.Command(exepath, args...).Start()
}

// RunOnboarding runs KiteOnboarding.exe in onboarding mode, which shows
// the plugin installer
func RunOnboarding() error {
	return run()
}

// RunLogin runs KiteOnboarding.exe in login mode, which shows the login step
// only.
func RunLogin() error {
	return run("--login-only")
}
