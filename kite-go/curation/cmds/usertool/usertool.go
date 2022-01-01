package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/curation"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func getenv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		log.Fatalf("expected environment variable %s is not set", name)
	}
	return val
}

func main() {
	var (
		name     string
		email    string
		password string
		list     bool
		updatepw bool
	)

	flag.StringVar(&name, "name", "", "username")
	flag.StringVar(&email, "email", "", "email")
	flag.StringVar(&password, "password", "", "password")
	flag.BoolVar(&list, "list", false, "list users")
	flag.BoolVar(&updatepw, "updatepw", false, "update password if true, create user if false")
	flag.Parse()

	db := curation.GormDB(getenv("CURATION_DB_DRIVER"), getenv("CURATION_DB_URI"))
	manager := community.NewUserManager(db, nil)

	if list {
		users, err := manager.List()
		if err != nil {
			log.Fatalln("error:", err)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		for _, user := range users {
			fmt.Fprintln(w, fmt.Sprintf("%s\t%s", user.Name, user.Email))
		}
		w.Flush()
		return
	}

	if updatepw {
		if email == "" || password == "" {
			log.Fatal("flags -email and -password are required for updating")
		}
		err := manager.UpdatePassword(email, password)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		return
	}

	if name == "" || email == "" || password == "" {
		log.Fatal("flags -name, -email and -password are required")
	}

	_, _, err := manager.Create(name, email, password, "")
	if err != nil {
		log.Fatalln("error:", err)
	}
}

// --

// setupSignupManagerWithDB takes a db and creates a new signup manager using it.
func setupSignupManagerWithDB(db gorm.DB) *community.SignupManager {
	manager := community.NewSignupManager(db)
	if err := manager.Migrate(); err != nil {
		log.Fatalln(err)
	}
	return manager
}
