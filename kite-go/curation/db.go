package curation

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	gorp "gopkg.in/gorp.v1"
)

const (
	// DatabaseLogin is the default username for curation database
	DatabaseLogin = "labelinguser"
	// DatabasePassword is the default password for curation database
	DatabasePassword = "XXXXXXX"
	// DatabaseHost is the default host for curation database
	DatabaseHost = "main.XXXXXXX.us-west-1.rds.amazonaws.com"
	// DatabasePort is the default port for curation database
	DatabasePort = 3306
	// DatabaseName is the default database name for curation database
	DatabaseName = "labeling"
)

var (
	// DatabaseURI is the default URI for curation database
	DatabaseURI = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", DatabaseLogin, DatabasePassword, DatabaseHost, DatabasePort, DatabaseName)
)

// GorpDB returns a *gorp.DbMap setup with curation tables and the provided driver and uri.
func GorpDB(driver, uri string) *gorp.DbMap {
	log.Println("curation gorp db using", driver, uri)
	db, err := sql.Open(driver, uri)
	if err != nil {
		log.Fatalf("error connecting to databse: %v\n", err)
	}

	var dialect gorp.Dialect
	switch driver {
	case "mysql":
		dialect = gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}
	case "sqlite3":
		dialect = gorp.SqliteDialect{}
	default:
		log.Fatalln("unrecognized db driver:", driver)
	}

	dbmap := OpenCodeExampleDb(db, dialect)
	if err != nil {
		log.Fatal(err)
	}
	dbmap.CreateTablesIfNotExists()
	return dbmap
}

// GormDB returns a gorm.DB object setup with the provider driver and uri.
func GormDB(driver, uri string) gorm.DB {
	log.Println("curation gorm db using", driver, uri)
	db, err := gorm.Open(driver, uri)
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)
	db.SingularTable(true)
	return db
}
