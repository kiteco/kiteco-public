// +build windows

package installid

// IDIfSet is not implemented for windows
func IDIfSet() (string, bool) {
	return "", false
}
