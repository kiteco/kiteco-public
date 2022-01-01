package envutil

import (
	"fmt"
	"os"
)

// MustSetenv sets the envirnment variable `key` to value `value` and `panic`s if there is an error.
func MustSetenv(key, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		panic(fmt.Errorf("error setting environment variable %s to %s: %v", key, value, err))
	}
}
