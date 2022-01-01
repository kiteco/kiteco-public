package fileutil

import (
	"fmt"
	"syscall"
)

// GetProperCasingPath returns the filepath as the os-native, case-sensitive value
// Microsoft's docs are at https://docs.microsoft.com/en-us/windows/desktop/api/fileapi/nf-fileapi-getlongpathnamew
// Testing showed that we need to pass the result of GetShortPathName into GetLongPathName
// to get a properly cased path
func GetProperCasingPath(filepath string) (string, error) {
	shortPath, err := getShortPathName(filepath)
	if err != nil {
		return filepath, err
	}

	longPath, err := getLongPathName(shortPath)
	if err != nil {
		return filepath, err
	}
	return longPath, nil
}

func getShortPathName(filepath string) (string, error) {
	pathPtr, err := syscall.UTF16PtrFromString(filepath)
	if err != nil {
		return filepath, err
	}

	// retrieve the required buffer length by passing nil and 0
	length, err := syscall.GetShortPathName(pathPtr, nil, 0)
	if err != nil || length == 0 {
		return filepath, fmt.Errorf("unable to retrieve short path name: %v", err)
	}

	buffer := make([]uint16, length)
	length, err = syscall.GetShortPathName(pathPtr, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 {
		return filepath, fmt.Errorf("unable to retrieve short path name: %v", err)
	}

	return syscall.UTF16ToString(buffer), nil
}

func getLongPathName(filepath string) (string, error) {
	pathPtr, err := syscall.UTF16PtrFromString(filepath)
	if err != nil {
		return filepath, err
	}

	// retrieve the required buffer length by passing nil and 0
	length, err := syscall.GetLongPathName(pathPtr, nil, 0)
	if err != nil || length == 0 {
		return filepath, fmt.Errorf("unable to retrieve short path name: %v", err)
	}

	buffer := make([]uint16, length)
	length, err = syscall.GetLongPathName(pathPtr, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 {
		return filepath, fmt.Errorf("unable to retrieve long path name: %v", err)
	}

	return syscall.UTF16ToString(buffer), nil
}
