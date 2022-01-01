package localfiles

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// FileDB builds a sqlx.DB object to use for the user-files database.
func FileDB(driver, uri string) *sqlx.DB {
	db, err := sqlx.Open(driver, uri)
	if err != nil {
		log.Fatal(err)
	}
	// Make sqlite connections serial
	if driver == "sqlite3" {
		db.SetMaxOpenConns(1)
	}
	db.SetConnMaxLifetime(time.Second * 60)
	return db
}
