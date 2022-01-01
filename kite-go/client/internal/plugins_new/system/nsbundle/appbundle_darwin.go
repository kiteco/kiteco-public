package nsbundle

// This file contains app bundle utilities unique to macOS.

import (
	"log"
	"os/exec"
	"strings"
)

// BundleLocations returns the absolute paths to the `.app` directories for a given bundle ID.
// TODO replace this with Cocoa C calls.
func BundleLocations(bundleID string) []string {
	out, err := exec.Command("mdfind", "kMDItemCFBundleIdentifier", "=", bundleID).Output()
	if err != nil {
		log.Println("could not detect bundle installations:", err)
		return nil
	}

	split := strings.Split(string(out), "\n")
	var valid []string
	for _, x := range split {
		if x != "" {
			valid = append(valid, x)
		}
	}
	return valid
}
