package event

import (
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/gorm"
)

// MetadataDB builds a gorm.DB object to use for the block metadata database.
func MetadataDB(driver, uri string) gorm.DB {
	log.Println("events metadata db using", driver, uri)
	db, err := gorm.Open(driver, uri)
	db.LogMode(true)
	if err != nil {
		log.Fatal(err)
	}
	db.SingularTable(true)
	db.DB().SetConnMaxLifetime(time.Second * 60)
	return db
}

// MetadataManager wraps the metadata db object.
type MetadataManager struct {
	db gorm.DB
}

// NewMetadataManager creates a new MetadataManager object.
func NewMetadataManager(db gorm.DB) *MetadataManager {
	return &MetadataManager{
		db: db,
	}
}

// Migrate migrates the metadata db.
func (mm *MetadataManager) Migrate() error {
	if err := mm.db.AutoMigrate(&Metadata{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	// Add user_id index
	if err := mm.db.Model(&Metadata{}).AddIndex("metadata_userid_idx", "user_id").Error; err != nil {
		return fmt.Errorf("error adding index: %v", err)
	}
	return nil
}

// Add creates new block metadata.
func (mm *MetadataManager) Add(metadata *Metadata) error {
	tx := mm.db.Begin()

	err := tx.FirstOrCreate(&metadata, Metadata{Filename: metadata.Filename}).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// Latest returns the up to n latest block metadata.
func (mm *MetadataManager) Latest(n, uid int64, stream string) ([]*Metadata, error) {
	var bm []*Metadata

	err := mm.db.Where(Metadata{UserID: uid, Stream: stream}).Order(
		"start desc").Limit(n).Find(&bm).Error

	if err != nil {
		return nil, err
	}

	return bm, nil
}

// Get returns up to n block metadata starting before timestamp ts.
// If the ts falls within a block, it returns the metadata for that block as well.
func (mm *MetadataManager) Get(n, uid int64, stream string, ts int64) ([]*Metadata, error) {
	var bm []*Metadata

	err := mm.db.Where(Metadata{UserID: uid, Stream: stream}).Where(
		"\"end\" < ?", ts).Order("start desc").Limit(n).Find(&bm).Error

	if err != nil {
		return nil, err
	}

	return bm, nil
}

// Delete removes the metadata from the db.
func (mm *MetadataManager) Delete(id int64) error {
	tx := mm.db.Begin()

	metadata := &Metadata{}

	if !tx.Where(Metadata{ID: id}).First(&metadata).RecordNotFound() {
		err := tx.Delete(&metadata).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

// -------------------------------

// Metadata holds all the metadata for block of events.
type Metadata struct {
	ID int64 // Primary key, by GORM convention

	Stream   string `valid:"required"` // Name of stream
	UserID   int64  `valid:"required"`
	Start    int64  `valid:"required"` // Timestamp of first event in batch
	End      int64  `valid:"required"` // Timestamp of last event in batch
	Count    int64  `valid:"required"` // Number of events in batch
	Size     int64  `valid:"required"` // Size of batch in bytes
	Filename string // Name of file stored in s3

	CreatedAt time.Time
}
