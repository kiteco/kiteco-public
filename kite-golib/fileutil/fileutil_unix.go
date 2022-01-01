// +build !windows

package fileutil

// GetProperCasingPath returns the filepath as the os-native, case-sensitive value
func GetProperCasingPath(filepath string) (string, error) {
	return filepath, nil
}
