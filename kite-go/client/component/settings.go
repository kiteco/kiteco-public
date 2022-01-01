package component

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/remotectrl"
)

// SettingsManager defines the functions to work with application settings
// Component interfaces must not depend on implementations
type SettingsManager interface {
	Core

	// Get returns the value associated with the key, and true/false depending
	// on whether the key was found.
	Get(key string) (string, bool)

	// GetBool returns the value associated with key, parsed as boolean and true/false depending
	// on whether the key was found.
	GetBool(key string) (bool, bool)

	// GetObj json deserializes the value of the provided key into the object
	GetObj(key string, obj interface{}) error

	// GetTime returns a time.Time of the provided key
	GetTime(key string) (time.Time, error)

	// GetDuration returns a time.Duration of the provided key
	GetDuration(key string) (time.Duration, error)

	// GetInt converts the value of the provided key into an int
	GetInt(key string) (int, error)

	// GetMaxFileSizeBytes returns max_file_size_kb in bytes as an int
	GetMaxFileSizeBytes() int

	// GetDeploymentID returns the set Kite Server deployment ID
	GetDeploymentID() string

	// Set will set the value for the provided key
	// it returns an error and a value suitable as http response code
	Set(key, value string) error

	// SetBool returns the value associated with key, parsed as boolean and true/false depending
	// on whether the key was found.
	SetBool(key string, value bool) error

	// SetObj json serializes the value of the provided object as the value for the key
	SetObj(key string, obj interface{}) error

	// Delete will remove the provided key from settings
	// It returns an error and a value suitable as http status code
	Delete(key string) error

	// AddNotificationTarget registers a new listener to settings updates
	AddNotificationTarget(target SettingsNotifier)

	// AddNotificationTargetKey registers a new listener to settings updates which listens to a specific key
	AddNotificationTargetKey(key string, callback func(newValue string))

	// Server returns the configured url of the backend server
	Server() string
	// SetServer sets a new url for the backend server, the changes are written to disl immediately.
	// Invalid URLs will be rejected with an error.
	// Also, an error is returned when an error occurred during save.
	SetServer(url string) error

	// HandleRemoteMessage allows remote setting of specific settings.
	remotectrl.Handler
}

// SettingsNotifier is used to notify listeners after settings change
// For a Component, implement the interface Settings from kite-go/client/component/interfaces.go:78 instead
// It will be automatically triggered without having to register your component in the settings manager
type SettingsNotifier interface {
	//Updated is called after a value was changed to a different value
	Updated(key, value string)
	//Deleted is called after a value was removed from the settings
	Deleted(key string)
}
