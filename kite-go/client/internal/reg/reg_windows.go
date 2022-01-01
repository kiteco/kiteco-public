package reg

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/sys/windows/registry"
)

// MachineID gets the value of the MachineID key
func MachineID() (string, error) {
	return get(registry.LOCAL_MACHINE, `Software\Kite`, "MachineID")
}

// IsDebug gets the value of the IsDebug key
func IsDebug() (string, error) {
	return get(registry.LOCAL_MACHINE, `Software\Kite`, "IsDebug")
}

// InstallPath gets the path where Kite was installed
func InstallPath() (string, error) {
	return get(registry.LOCAL_MACHINE, `Software\Kite\AppData`, "InstallPath")
}

// WasVisible returns the value of WasVisible
func WasVisible() (bool, error) {
	val, err := get(registry.CURRENT_USER, `Software\Kite\AppData`, "WasVisible")
	return val == "1", err
}

// SetWasVisible sets the value of WasVisible
func SetWasVisible(val bool) error {
	var setVal string
	if val {
		setVal = "1"
	} else {
		setVal = "0"
	}
	return set(registry.CURRENT_USER, `Software\Kite\AppData`, "WasVisible", setVal)
}

// UpdateHKCURun updates the registry entry that controls starting kite on startup.
func UpdateHKCURun() error {
	installdir, err := InstallPath()
	if err != nil {
		log.Println("not updating Software\\Microsoft\\Windows\\CurrentVersion\\Run, could not find install path")
		return err
	}

	return set(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "Kite",
		fmt.Sprintf("\"%s\" --system-boot", filepath.Join(installdir, "kited.exe")))
}

// RemoveHKCURun removes the registry entry that starts kite on startup.
func RemoveHKCURun() error {
	if val, err := get(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "Kite"); err != nil || val == "" {
		return err
	}
	return deleteValue(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "Kite")
}

// TrayIconHandle gets the value of the TrayIconHandle key
func TrayIconHandle() (uintptr, error) {
	s, err := get(registry.CURRENT_USER, `Software\Kite\AppData`, "LastTrayHwnd")
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uintptr(i), nil
}

// SetTrayIconHandle sets the value of the TrayIconHandle key
func SetTrayIconHandle(h uintptr) error {
	return set(registry.CURRENT_USER, `Software\Kite\AppData`, "LastTrayHwnd", fmt.Sprintf("%d", h))
}

// get looks up a string-valued name in the Kite registry key.
func get(key registry.Key, path, item string) (string, error) {
	k, err := registry.OpenKey(key, path, registry.QUERY_VALUE)
	if err != nil {
		return "", errors.Errorf("error opening key %s: %v", path, err)
	}
	defer k.Close()

	s, _, err := k.GetStringValue(item)
	if err != nil {
		return "", errors.Errorf("error getting value for %s in %v: %v", item, key, err)
	}
	return s, err
}

// set set a string-valued name in the Kite registry key.
func set(key registry.Key, path, item, value string) error {
	// create the key if it does not exist
	k, _, err := registry.CreateKey(key, path, registry.ALL_ACCESS)
	if err != nil {
		// note that CreateKey does not return an error if the key already exists
		return errors.Errorf("error creating/opening key %s: %v", path, err)
	}
	defer k.Close()
	return k.SetStringValue(item, value)
}

// deleteValue removes a value in the Kite registry key.
func deleteValue(key registry.Key, path, item string) error {
	k, err := registry.OpenKey(key, path, registry.ALL_ACCESS)
	if err != nil {
		return errors.Errorf("error opening key %s: %v", path, err)
	}
	defer k.Close()

	return k.DeleteValue(item)
}
