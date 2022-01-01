package community

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
)

// DB builds a gorm.DB object to use for the community database.
func DB(driver, uri string) gorm.DB {
	db, err := gorm.Open(driver, uri)
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)
	db.SingularTable(true)
	db.DB().SetConnMaxLifetime(time.Second * 60)
	return db
}
