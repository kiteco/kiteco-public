package main

import (
	"flag"
	"log"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
)

func main() {
	var email, host string
	flag.StringVar(&email, "email", "", "email to invite to Kite")
	flag.StringVar(&host, "host", "", "host for download link of Kite")
	flag.Parse()

	if email == "" {
		log.Fatalln("Please specify an email address to invite to Kite.")
	}
	if host == "" {
		log.Fatalln("Please specify the host.")
	}

	db := community.DB(envutil.MustGetenv("COMMUNITY_DB_DRIVER"), envutil.MustGetenv("COMMUNITY_DB_URI"))
	app := community.NewApp(db, community.NewSettingsManager(), nil)

	code, err := app.Signups.Invite(email, host)
	if err != nil {
		log.Fatalf("Error inviting email: %v", err)
	}

	log.Printf("Successfully invited %s. Invite code: %s", email, code)
}
