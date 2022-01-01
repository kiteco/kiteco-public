package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/mitchellh/cli"
)

// ensureDownloaded ensures that the update package is downloaded at localManager.downloadTargetPath
func ensureDownloaded(ui cli.Ui, local *localManager, updater *updateManager, remoteVersion Version, publicKey string, tracker *errorTracker, silentUpdate bool) error {
	target := local.downloadTargetPath(remoteVersion)
	targetDir := filepath.Dir(target)

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return errors.Errorf("error creating %s: %s", targetDir, err.Error())
	}

	// Check if file already exists and is valid
	if _, err := os.Stat(target); err == nil || os.IsExist(err) {
		ui.Info(fmt.Sprintf("%s already exists, validating...", target))
		err = validateSha256Checksum(target, remoteVersion.Sha256Checksum)
		if err == nil {
			ui.Info(fmt.Sprintf("checksums match, continuing with existing file"))
			return nil
		}

		ui.Info(fmt.Sprintf("checksum doesn't match, downloading again..."))
	}

	err := os.Remove(target)
	if err != nil && !os.IsNotExist(err) {
		return errors.Errorf("error removing %s: %s", target, err.Error())
	}

	onProgress := progressPrinter(ui)
	if silentUpdate {
		onProgress = nil
	}

	err = updater.downloadUpdate(target, remoteVersion, onProgress)
	if err != nil {
		defer os.Remove(target)
		tracker.addDownloadError(remoteVersion.Version)
		return errors.Errorf("error downloading %s: %s", err, err.Error())
	}

	ui.Info("verifying checksum")
	err = validateSha256Checksum(target, remoteVersion.Sha256Checksum)
	if err != nil {
		defer os.Remove(target)
		tracker.addValidationError(remoteVersion.Version)
		ui.Info("unable to validate downloaded package, cannot continue")
		return errors.Errorf("error validating checksum of downloaded update file: %s", err.Error())
	}

	sig, err := remoteVersion.SignatureBytes()
	if err != nil {
		defer os.Remove(target)
		return errors.Errorf("error retrieving signature: %s", err.Error())
	}

	ui.Info("validating signature")
	if err = validateSignature(target, publicKey, sig); err != nil {
		tracker.addValidationError(remoteVersion.Version)
		defer os.Remove(target)
		return errors.Errorf("error validating signature: %s", err.Error())
	}

	return nil
}

// install executes the self-extracting installer file and update the current link
func install(local *localManager, remoteVersion Version) error {
	target := local.downloadTargetPath(remoteVersion)
	defer os.Remove(target)

	cmd := exec.Command("/bin/sh", target)
	cmd.Dir = filepath.Dir(target)

	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error executing self-extracting script",
			Err:    err,
			Output: string(buf),
		}
	}

	installDir := local.installDirPath(remoteVersion.Version)

	_, err = os.Stat(installDir)
	if err != nil && os.IsNotExist(err) {
		return errors.Errorf("could not find install path %s after installation: %s", installDir, err.Error())
	}

	// link latest directory to <prefix>/current to switch as quickly as possible
	// we want to avoid a situation when the update is extracting and a user starts kited.
	// in this case the user would start kite from a not-yet unpacked installation
	err = local.updateCurrentLink(remoteVersion.Version)
	if err != nil {
		return errors.Errorf("failed to update symbolic link to the latest version: %s", err.Error())
	}

	return nil
}

// installSystemData installs a new set of system data and enables the service and protocol handler afterwards
func installSystemData(ui cli.Ui, actionType string) int {
	err := inflateBindataFiles(ui)
	if err != nil {
		ui.Error(fmt.Sprintf("error installing system files: %s", err.Error()))
		rollbarError("error installing system files", actionType, err)
		// recover, the service must not remain stopped
		err = enableAndStartUpdaterService()
		if err != nil {
			rollbarError("error restarting updater service after install failure", actionType, err)
		}
		return 1
	}

	// reload changed systemd units to make sure that enable below will find the services
	_ = systemctlReloadDaemon()

	// the service files were updated. Now refresh and start our service
	// a failed update of our systemd services no longer is an error,
	// kited is capable to update itself
	ui.Info("activating kite-updater systemd service")
	err = enableAndStartUpdaterService()
	if err != nil {
		ui.Warn("error enabling kite-updater.timer. " + err.Error())
		rollbarError("failed to enable updater service", actionType, err)
		// don't rollback, because installs without X11 may not have xdg-open installed
	}

	// the service files were updated. Now refresh and enable our autostart service
	// we're not starting the service, because the install command starts kited on its own
	// and the update command triggers a restart with kited's HTTP endpoint.
	ui.Info("activating kite-autostart systemd service")
	err = enableAutostartService()
	if err != nil {
		ui.Warn("error enabling kite-autostart. " + err.Error())
		rollbarError("failed to enable autostart service", actionType, err)
	}

	// update the registration of our custom protocol handler
	ui.Info("registering kite:// protocol handler")
	err = registerKiteProtocolHandler()
	if err != nil {
		ui.Error("error installing kite protocol handler. " + err.Error())
		rollbarError("failed to enable kite:// protocol handler", actionType, err)
		// don't rollback, because installs without X11 may not have xdg-open installed
	}
	return 0
}

func rollbarError(errMsg, actionType string, err error) {
	switch e := err.(type) {
	case cmdError:
		rollbar.Error(errors.Errorf(errMsg), actionType, e.Msg, e.Err.Error(), e.Output, uname())
	default:
		rollbar.Error(errors.Errorf(errMsg), actionType, err.Error(), uname())
	}
}

func uname() string {
	cmd := exec.Command("uname", "-a")
	buf, _ := cmd.CombinedOutput()
	return string(buf)
}

// removeOldVersions removes outdated versions from disk. We're not keeping old versions of kited around.
func removeOldVersions(local *localManager, currentVersion Version) {
	if installedVersions, err := local.installedVersions(); err == nil {
		for _, v := range installedVersions {
			if v != currentVersion.Version {
				if err = local.uninstallVersion(v); err != nil {
					log.Printf("error uninstalling version %s: %s", v, err.Error())
				}
			}
		}
	}
}

// validateSha256Checksum returns an error if the sha256 checksum of the given
// file is not matching the expected hexadecimal hash value
func validateSha256Checksum(filePath string, expected string) error {
	in, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer in.Close()

	h := sha256.New()
	if _, err := io.Copy(h, in); err != nil {
		return err
	}

	if actual := hex.EncodeToString(h.Sum(nil)); actual != expected {
		var size int64 = -1
		if stat, err := os.Stat(filePath); err == nil {
			size = stat.Size()
		}

		return errors.Errorf("sha256 hash of file %s isn't matching expected value. %s != %s (expected), %d bytes retrieved", filePath, actual, expected, size)
	}

	return nil
}

func tildify(path string) string {
	homeDir := os.ExpandEnv("$HOME")
	return strings.Replace(path, homeDir, "~", -1)
}

func inflateBindataFiles(ui cli.Ui) error {
	type templateData struct {
		HomeDir string
		Version string
	}

	homeDir := os.ExpandEnv("$HOME")
	tmplData := templateData{
		HomeDir: homeDir,
		Version: version,
	}

	assetNames := AssetNames()
	sort.Strings(assetNames)

	for _, fn := range assetNames {
		switch {
		case filepath.Ext(fn) == ".template":
			var buf bytes.Buffer
			if err := renderText(&buf, fn, tmplData); err != nil {
				return errors.Errorf("error installing ~/%s: %s", fn, err.Error())
			}

			// construct full path, trim .template extension
			fullPath := filepath.Join(homeDir, strings.TrimSuffix(fn, filepath.Ext(fn)))
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err != nil {
				return errors.Errorf("error installing %s: %s", tildify(fullPath), err.Error())
			}

			err = ioutil.WriteFile(fullPath, buf.Bytes(), 0600)
			if err != nil {
				return errors.Errorf("error installing %s: %s", tildify(fullPath), err.Error())
			}

			ui.Info(fmt.Sprintf("installed %s", tildify(fullPath)))

		default:
			err := RestoreAsset(homeDir, fn)
			if err != nil {
				return errors.Errorf("error installing ~/%s: %s", fn, err.Error())
			}

			ui.Info(fmt.Sprintf("installed ~/%s", fn))
		}
	}

	return nil
}

func removeBindataFiles(ui cli.Ui) error {
	homeDir := os.ExpandEnv("$HOME")

	assetNames := AssetNames()
	sort.Strings(assetNames)

	for _, fn := range assetNames {
		if filepath.Ext(fn) == ".template" {
			fn = strings.TrimSuffix(fn, filepath.Ext(fn))
		}

		fullPath := filepath.Join(homeDir, fn)
		if !exists(fullPath) {
			continue
		}

		err := os.RemoveAll(fullPath)
		if err != nil {
			ui.Error(fmt.Sprintf("error removing ~/%s: %s", fn, err.Error()))
			continue
		}

		ui.Info(fmt.Sprintf("removed ~/%s", fn))
	}

	return nil
}

type cmdError struct {
	Msg    string
	Err    error
	Output string
}

func (c cmdError) Error() string {
	return fmt.Sprintf("%s, output: '%s'", c.Err.Error(), c.Output)
}

func updateSystemData(updaterPath string) error {
	cmd := exec.Command(updaterPath, "update-system-data")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error running update-system-data",
			Err:    err,
			Output: string(buf),
		}
	}
	return nil
}

func stopAndDisableUpdaterService() error {
	cmd := exec.Command("systemctl", "--user", "disable", "--now", "kite-updater.timer")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error disabling kite-updater.timer",
			Err:    err,
			Output: string(buf),
		}
	}

	// "man systemctl" notes that "disable" is doing the equivalent of "daemon-reload" after completing the operation.
	// we're doing this as a safeguard, but it's probably not needed
	return systemctlReloadDaemon()
}

func enableAndStartUpdaterService() error {
	cmd := exec.Command("systemctl", "--user", "enable", "--now", "kite-updater.timer")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error enabling kite-updater.timer",
			Err:    err,
			Output: string(buf),
		}
	}

	// "man systemctl" notes that "enable" is doing the equivalent of "daemon-reload" after completing the operating.
	// we're doing this as a safeguard, but it's probably not needed
	return systemctlReloadDaemon()
}

func stopAndDisableAutostartService() error {
	cmd := exec.Command("systemctl", "--user", "disable", "--now", "kite-autostart")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error disabling kite-autostart",
			Err:    err,
			Output: string(buf),
		}
	}

	// "man systemctl" notes that "disable" is doing the equivalent of "daemon-reload" after completing the operation.
	// we're doing this as a safeguard, but it's probably not needed
	return systemctlReloadDaemon()
}

func enableAutostartService() error {
	cmd := exec.Command("systemctl", "--user", "enable", "kite-autostart")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error enabling kite-autostart",
			Err:    err,
			Output: string(buf),
		}
	}

	// "man systemctl" notes that "enable" is doing the equivalent of "daemon-reload" after completing the operating.
	// we're doing this as a safeguard, but it's probably not needed
	return systemctlReloadDaemon()
}

func startAutostartService() error {
	cmd := exec.Command("systemctl", "--user", "start", "kite-autostart")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error starting kite-autostart",
			Err:    err,
			Output: string(buf),
		}
	}
	return nil
}

func systemctlReloadDaemon() error {
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error running 'systemctl --user daemon-reload'",
			Err:    err,
			Output: string(buf),
		}
	}
	return nil
}

func registerKiteProtocolHandler() error {
	cmd := exec.Command("xdg-mime", "default", "kite-copilot.desktop", "x-scheme-handler/kite")
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return cmdError{
			Msg:    "error running 'xdg-mime default kite-copilot.desktop x-scheme-handler/kite'",
			Err:    err,
			Output: string(buf),
		}
	}
	return nil
}

// launchKite first tries to launch kited with systemd and then tries to run it via commandline
func launchKite(local *localManager) error {
	// first, try to launch with systemd
	err := startAutostartService()
	if err == nil {
		return nil
	}

	// fallback to launch the kited via commandline
	kiteLaunchScript := filepath.Join(local.basePath, "kited")
	cmd := exec.Command(kiteLaunchScript)
	err = cmd.Start()
	if err != nil {
		return err
	}
	cmd.Process.Release()
	return nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func isTTY(file *os.File) bool {
	fi, _ := file.Stat()
	return fi.Mode()&os.ModeCharDevice != 0
}

// progressPrinter returns a function which prints the current progress defined by received and total
// with a TTY progress will be printed on a single line
// without a TTY progress will be printed on separate lines
func progressPrinter(ui cli.Ui) func(received, total int64) {
	if isTTY(os.Stdout) {
		// print progress on the current line when using a TTY
		var lastReceived int64 = -1
		return func(received, total int64) {
			// only print if it's at least a 0.1% increment
			if float64(received-lastReceived)/float64(total) > 0.001 {
				var prefix string
				if pUI, ok := ui.(*cli.PrefixedUi); ok {
					prefix = pUI.InfoPrefix
				}
				// use fmt instead of ui because ui isn't supporting \r
				// right align the progress to always print lines of the same length
				// \r overwrites the current line, so it has to be at least of the same length as the previously printed line
				fmt.Printf("\r%sDownloading Kite: %6.1f%% of %s", prefix, (float64(received)/float64(total))*100, humanize.IBytes(uint64(total)))
				lastReceived = received
			}
			// print a newline after the download finished
			if received >= total {
				fmt.Print("\n")
			}
		}
	}

	// print received and total numbers when not on a TTY
	// layout: "Download <received bytes>/<total bytes>", this is used by kite-connect-js
	// we're not using "ui" here to avoid the additional prefix
	return func(received, total int64) {
		fmt.Printf("Download: %d/%d\n", received, total)
	}
}
