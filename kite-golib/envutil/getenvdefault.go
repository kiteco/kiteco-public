package envutil

import (
	"log"
	"os"
	"strconv"
)

// GetenvDefault gets the value of an environment variable, or returns the
// specified default value if that variable is not set.
func GetenvDefault(name, defaultValue string) string {
	val, found := os.LookupEnv(name)
	if !found {
		return defaultValue
	}
	return val
}

// GetenvDefaultInt gets an environment variable as an int, or else returns the default
func GetenvDefaultInt(name string, defaultVal int) int {
	val, found := os.LookupEnv(name)
	if !found {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("environment variable %s should be an integer: %v", name, err)
	}
	return intVal
}
