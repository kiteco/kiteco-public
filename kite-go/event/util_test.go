package event

import _ "github.com/mattn/go-sqlite3"

func createTestManager() *MetadataManager {
	mdb := MetadataDB("sqlite3", ":memory:")
	mm := NewMetadataManager(mdb)
	mm.Migrate()
	return mm
}
