// +build linux standalone

package messagebox

import "log"

func show(opts Options) error {
	log.Println(opts.Text)
	return nil
}

func showAlert(opts Options) error {
	return show(opts)
}

func showWarning(opts Options) error {
	return show(opts)
}
