package release

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// MockMetadataManager provides hardcoded metadata to test client updates.
type MockMetadataManager struct{}

// NewMockMetadataManager initializes MockMetadataManager.
func NewMockMetadataManager() MetadataManager {
	return &MockMetadataManager{}
}

// Migrate doesn't have to do anything for our hardcoded data, but exists to
// satisfy the MetadataManager interface.
func (mmm *MockMetadataManager) Migrate() error {
	return nil
}

// Create doesn't have to do anything for our hardcoded data, but exists to
// satisfy the MetadataManager interface.
func (mmm *MockMetadataManager) Create(a MetadataCreateArgs) (*Metadata, error) {
	return nil, nil
}

// LatestCanary returns this hardcoded metadata, so that we can test clients upgrading to version N+1.
func (mmm *MockMetadataManager) LatestCanary(client Platform) (*Metadata, error) {
	if client == Mac {
		return &Metadata{
			Client:       Mac,
			Version:      "0.99999999.0",
			CreatedAt:    time.Now(),
			DSASignature: "MCwCFFEi1I9txbV9EkbDY5jhu4XnFb2ZAhQaj8vcD+yzwJEl/vffuNKWs4bJYQ==",
			GitHash:      "1aa5e8898b4832a3ec39c97f66f6e9406900c24e",
		}, nil
	} else if client == Windows {
		return &Metadata{
			Client:            Mac,
			Version:           "1.9999.9999.0",
			CreatedAt:         time.Now(),
			DSASignature:      "",
			GitHash:           "",
			ReleasePercentage: 100,
		}, nil
	} else if client == Linux {
		return &Metadata{
			Client:       Linux,
			Version:      "2.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else {
		return nil, errors.New("not implemented for platform")
	}
}

// LatestNonCanary returns this hardcoded metadata, so that we can test clients upgrading to version N+1.
func (mmm *MockMetadataManager) LatestNonCanary(client Platform) (*Metadata, error) {
	if client == Mac {
		return &Metadata{
			Client:       Mac,
			Version:      "0.99999999.0",
			CreatedAt:    time.Now(),
			DSASignature: "MCwCFFEi1I9txbV9EkbDY5jhu4XnFb2ZAhQaj8vcD+yzwJEl/vffuNKWs4bJYQ==",
			GitHash:      "1aa5e8898b4832a3ec39c97f66f6e9406900c24e",
		}, nil
	} else if client == Windows {
		return &Metadata{
			Client:            Mac,
			Version:           "1.9999.9999.1",
			CreatedAt:         time.Now(),
			DSASignature:      "",
			GitHash:           "",
			ReleasePercentage: 25,
		}, nil
	} else if client == Linux {
		return &Metadata{
			Client:       Linux,
			Version:      "2.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else {
		return nil, errors.New("not implemented for platform")
	}
}

// FromVersion returns this hardcoded metadata, so that we can test clients upgrading
// to version N+1.
func (mmm *MockMetadataManager) FromVersion(client Platform, version string) (*Metadata, error) {
	if client == Mac {
		return &Metadata{
			Client:       Mac,
			Version:      "0.99999999.0",
			CreatedAt:    time.Now(),
			DSASignature: "MCwCFFEi1I9txbV9EkbDY5jhu4XnFb2ZAhQaj8vcD+yzwJEl/vffuNKWs4bJYQ==",
			GitHash:      "1aa5e8898b4832a3ec39c97f66f6e9406900c24e",
		}, nil
	} else if client == Windows {
		return &Metadata{
			Client:       Mac,
			Version:      "1.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else if client == Linux {
		return &Metadata{
			Client:       Linux,
			Version:      "2.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else {
		return nil, errors.New("not implemented for platform")
	}
}

// FromID returns this hardcoded metadata, so that we can test clients upgrading
// to version N+1.
func (mmm *MockMetadataManager) FromID(client Platform, id int64) (*Metadata, error) {
	if client == Mac {
		return &Metadata{
			Client:       Mac,
			Version:      "0.99999999.0",
			CreatedAt:    time.Now(),
			DSASignature: "MCwCFFEi1I9txbV9EkbDY5jhu4XnFb2ZAhQaj8vcD+yzwJEl/vffuNKWs4bJYQ==",
			GitHash:      "1aa5e8898b4832a3ec39c97f66f6e9406900c24e",
		}, nil
	} else if client == Windows {
		return &Metadata{
			Client:       Mac,
			Version:      "1.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else if client == Linux {
		return &Metadata{
			Client:       Linux,
			Version:      "2.9999.9999.0",
			CreatedAt:    time.Now(),
			DSASignature: "",
			GitHash:      "",
		}, nil
	} else {
		return nil, errors.New("not implemented for platform")
	}
}

// NextVersion shouldn't really ever be used, but it returns the correct value
// just in case.
func (mmm *MockMetadataManager) NextVersion(client Platform) (string, error) {
	if client == Mac {
		return "0.99999999.1", nil
	} else if client == Windows {
		return "1.9999.9999.1", nil
	} else if client == Linux {
		return "2.9999.9999.1", nil
	} else {
		return "", errors.New("not implemented for platform")
	}
}

// Publish publishes the client-version to a percentage of users.
// It's a no-op for this mock manager.
func (mmm *MockMetadataManager) Publish(client Platform, version string, percentage uint8) error {
	return nil
}

// CreateDelta adds a delta update to the db for the given client and from->to versions.
// It's a no-op for this mock manager.
func (mmm *MockMetadataManager) CreateDelta(client Platform, fromVersion, toVersion, dsaSignature string) (*Delta, error) {
	return nil, nil
}

// DeltasToVersion returns deltas update to the given version.
// It's a no-op for this mock manager.
func (mmm *MockMetadataManager) DeltasToVersion(client Platform, version string) ([]*Delta, error) {
	return nil, nil
}
