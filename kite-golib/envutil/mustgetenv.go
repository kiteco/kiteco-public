package envutil

import (
	"log"
	"os"
	"strconv"
)

// MustGetenv gets the value of an environment variable, or exits if it has no value.
func MustGetenv(name string) string {
	val, found := os.LookupEnv(name)
	if !found {
		log.Fatalf("Environment variable %s is required but not set", name)
	}
	return val
}

// MustGetenvInt gets an environment variable as an int, or else exits with an error message
func MustGetenvInt(name string) int {
	val, found := os.LookupEnv(name)
	if !found {
		log.Fatalf("environment variable %s is required but not set", name)
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("environment variable %s should be an integer: %v", name, err)
	}
	return intVal
}

// MustGetenvInt64 gets an environment variable as an int64, or else exits with an error message
func MustGetenvInt64(name string) int64 {
	val, found := os.LookupEnv(name)
	if !found {
		log.Fatalf("environment variable %s is required but not set", name)
	}
	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		log.Fatalf("environment variable %s should be an integer: %v", name, err)
	}
	return intVal
}
