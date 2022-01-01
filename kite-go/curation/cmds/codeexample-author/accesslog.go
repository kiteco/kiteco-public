package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/curation"
)

const defaultAccessTimeout = 60 // in seconds

type accessManager struct {
	db      gorm.DB
	timeout int64 // access lock timeout, in seconds
}

func newAccessManager(db gorm.DB) *accessManager {
	return &accessManager{db: db, timeout: defaultAccessTimeout}
}

// Migrate will auto-migrate relevant tables in the db.
func (a *accessManager) Migrate() error {
	if err := a.db.AutoMigrate(&curation.PackageAccess{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	return nil
}

// Record records the current time as the most recent access time for the given package.
func (a *accessManager) record(language, pkg, user string) error {
	pkgKey := fmt.Sprintf("%s:%s", language, pkg)
	record := curation.PackageAccess{
		Package:   pkgKey,
		User:      user,
		Timestamp: time.Now().Unix(),
	}
	var access curation.PackageAccess
	if err := a.db.Where("Package=?", pkgKey).First(&access).Error; err != nil {
		if err == gorm.RecordNotFound {
			if err := a.db.Create(&record).Error; err != nil {
				return fmt.Errorf("error while creating PackageAccess entry: %v", err)
			}
			return nil
		}
		return fmt.Errorf("error while checking for existing PackageAccess entry: %v", err)
	}

	newVals := map[string]interface{}{
		"User":      user,
		"Timestamp": time.Now().Unix(),
	}

	if err := a.db.Table("PackageAccess").Where("Package=?", pkgKey).Updates(newVals).Error; err != nil {
		return fmt.Errorf("error while updating PackageAccess entry: %v", err)
	}

	return nil
}

// CurrentAccessor returns the name of the user currently accessing the package, or the empty
// string if there are no users accessing the package.
func (a *accessManager) currentAccessor(language, pkg string) string {
	accessLock, err := a.currentAccessLock(language, pkg)
	if err != nil {
		log.Printf("error getting access lock in currentAccessor: %v\n", err)
	}
	if accessLock == nil {
		return ""
	}
	return accessLock.UserEmail
}

type accessLock struct {
	UserEmail  string `json:"userEmail"`
	Expiration int64  `json:"expiration"`
}

func (a *accessManager) currentAccessLock(language, pkg string) (*accessLock, error) {
	now := time.Now().Unix()
	var record curation.PackageAccess
	name := language + ":" + pkg
	err := a.db.Where("Package=?", name).Limit(1).Find(&record).Error
	if err == gorm.RecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while getting access logs: %v", err)
	}
	if record.Timestamp+a.timeout > now {
		// keep these debug logs until we're sure there are no remaining concurrency issues:
		log.Printf("currently lock for %s held by %s\n", name, record.User)
		log.Println("last accessed at", record.Timestamp)
		log.Println("so it expires at", record.Timestamp+a.timeout)
		return &accessLock{
			UserEmail:  record.User,
			Expiration: record.Timestamp + a.timeout,
		}, nil
	}
	log.Println("currently no lock held for", name)
	return nil, nil
}

func (a *accessManager) acquireAccessLock(language, pkg, userEmail string) (*accessLock, error) {
	now := time.Now().Unix()
	currentLock, err := a.currentAccessLock(language, pkg)
	if err != nil {
		return nil, err
	}
	if currentLock != nil && currentLock.UserEmail != userEmail {
		log.Println("existing lock held")
		return currentLock, nil
	}
	log.Println("acquiring new lock for", userEmail)
	// There is a race condition here, when another user acquires the lock in
	// between us checking the lock and us updating the access logs.
	err = a.record(language, pkg, userEmail)
	if err != nil {
		return nil, fmt.Errorf("error while updating access logs: %v", err)
	}
	return &accessLock{
		UserEmail:  userEmail,
		Expiration: now + a.timeout, // TODO get actual expiration don't fudge it from `now`
	}, nil
}

// AllAccessors returns a map from package names to the name of the user currently accessing
// that package. Packages that are not currently being accessed will not appear in the map.
func (a *accessManager) allAccessors() map[string]string {
	now := time.Now().Unix()
	var records []curation.PackageAccess
	if err := a.db.Find(&records).Error; err != nil {
		return nil
	}

	accessors := make(map[string]string)
	for _, record := range records {
		if record.Timestamp >= now-a.timeout {
			accessors[record.Package] = record.User
		}
	}
	return accessors
}
