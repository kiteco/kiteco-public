package localcode

import (
	"github.com/kiteco/kiteco/kite-go/core"
)

// Context defines the API that is used by user-node based services to retrieve localcode artifacts
type Context interface {
	// ArtifactForFile returns a loaded artifact that can serve the provided file or directory path.
	// This object should not be stored or persisted anywhere.
	// Always request a new Artifact as updates may have been made by the localcode framework.
	ArtifactForFile(path string) (interface{}, error)

	// RequestForFile returns a loaded artifact that can serve the provided filename. If one is not found
	// it will request one from the worker. This object should not be stored or persisted anywhere. Always
	// request a new Artifact as updates may have been made by the localcode framework.
	// trackExtra may be logged as tracking data in case of any error.
	RequestForFile(filename string, trackExtra interface{}) (interface{}, error)

	// AnyArtifact returns the most general artifact that can be found. This object
	// should not be stored or persisted anywhere. Always request a new Artifact as updates may have
	// been made by the localcode framework.
	AnyArtifact() (interface{}, error)

	// Status returns a list of Status objects for the provided user/machine
	Status(n int) []*Status

	// StatusResponse constructs the response for requests for local code index status
	StatusResponse(n int) *StatusResponse

	// Cleanup cleans up any state UserContext may be holding on to via artifacts
	Cleanup() error

	// FileDriver returns the latest available unified driver for the file
	// If more than one editor was used to edit the file, then the last used file driver is returned
	LatestFileDriver(unixFilepath string) (core.FileDriver, error)
}
