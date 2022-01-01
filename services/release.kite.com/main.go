package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/release"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type cmd struct {
	name    string
	argNum  int
	usage   string
	example string
	run     func(release.MetadataManager, []string)

	managerImpl release.MetadataManager
}

func (c *cmd) printHelp() {
	if c.example == "" {
		fmt.Printf("usage:\n  %s\n", c.usage)
	} else {
		fmt.Printf("usage:\n  %s\nexample:\n  %s\n", c.usage, c.example)
	}
}

// This is duplicated in kite-go/cmds/release/main.go for development
var cmdServer = &cmd{
	name:   "server",
	argNum: 0,
	usage:  "release server",
	run: func(manager release.MetadataManager, args []string) {
		router := mux.NewRouter()
		release.NewServer(router, manager, release.DefaultPlatforms)
		logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds)
		neg := negroni.New(
			midware.NewRecovery(),
			midware.NewLogger(logger),
			negroni.Wrap(router),
		)
		err := http.ListenAndServe(":9093", neg)
		if err != nil {
			log.Println(err)
		}
	},
}

func main() {
	flag.Parse()
	args := flag.Args()

	cmds := []*cmd{cmdServer}

	if len(args) > 0 {
		for _, c := range cmds {
			if c.name == args[0] {
				args = args[1:]

				if len(args) > 0 && args[0] == "help" {
					c.printHelp()
					os.Exit(0)
				}
				if c.argNum != len(args) {
					c.printHelp()
					os.Exit(1)
				}

				var manager release.MetadataManager
				switch c.managerImpl.(type) {
				case *release.MockMetadataManager:
					manager = release.NewMockMetadataManager()
				default:
					dbDriver := envutil.MustGetenv("RELEASE_DB_DRIVER")
					dbURI := envutil.MustGetenv("RELEASE_DB_URI")
					dbEnv, _ := os.LookupEnv("RELEASE_DB_ENV")
					readsPublic := dbEnv != "staging"

					manager = release.NewMetadataManager(dbDriver, dbURI, readsPublic)
					err := manager.Migrate()
					if err != nil {
						log.Fatalln("failed to migrate release database:", err)
					}
				}
				c.run(manager, args)
				os.Exit(0)
			}
		}
	}

	for _, c := range cmds {
		c.printHelp()
		fmt.Println("")
	}
	os.Exit(1)
}
