// +build !darwin

package startup

func mode() Mode {
	return ManualLaunch
}

func reset() {
	return
}
