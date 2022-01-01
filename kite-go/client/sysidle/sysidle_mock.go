// +build !darwin

package sysidle

func sysIdle() bool {
	return false
}
