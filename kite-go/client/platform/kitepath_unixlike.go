// +build !windows

package platform

import "os"

// kiteRoot returns the directory containing kite configuration and session files.
func kiteRoot() string {
	return os.ExpandEnv("$HOME/.kite")
}
