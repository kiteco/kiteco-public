package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/release"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// - Named inputs & outputs, using environment variables

// Param is a name parameter
type Param string

// Params
const (
	NumDeltas Param = "_NUM_DELTAS"

	ID                Param = "_ID"
	Platform          Param = "_PLATFORM"
	Version           Param = "_VERSION"
	FromVersion       Param = "_FROM_VERSION"
	CreatedAt         Param = "_TIMESTAMP"
	Signature         Param = "_SIGNATURE"
	GitHash           Param = "_GIT_HASH"
	ReleasePercentage Param = "_CANARY_PERCENTAGE"

	ToVersion Param = Version
)

func deltaSuffix(i int) string {
	return fmt.Sprintf("_DELTA_%d", i)
}

func getParamSuffix(p Param, suffix string) interface{} {
	val, ok := os.LookupEnv(string(p) + suffix)
	if !ok {
		log.Printf("environment variable %s not set", p)
		os.Exit(1)
	}

	var res interface{}
	var err error
	switch p {
	case NumDeltas:
		i, err2 := strconv.Atoi(val)
		if err2 != nil || i < 0 {
			err = errors.New("must be a natural number")
		}
		res = i
	case Platform:
		res, err = release.ParsePlatform(val)
	case FromVersion, Version, GitHash:
		if val == "" {
			err = errors.New("must be nonempty")
		}
		res = val
	case Signature:
		res = val
	case ReleasePercentage:
		pct, err2 := strconv.ParseUint(val, 10, 8)
		if err2 != nil || pct > 100 {
			err = errors.New("must be a number between 0 and 100")
		}
		res = uint8(pct)
	default:
		panic("unhandled param")
	}
	if err != nil {
		log.Printf("invalid %s=\"%s\" %s", p, val, err)
		os.Exit(1)
	}
	return res
}

// GetParam gets the provided Param, of the appropriate type
func GetParam(p Param) interface{} {
	return getParamSuffix(p, "")
}

// GetDeltaParam gets the provided indexed Param a set of Deltas
func GetDeltaParam(i int, p Param) interface{} {
	return getParamSuffix(p, deltaSuffix(i))
}

// Serialize outputs a set of named Params corresponding to a Metadata
func Serialize(m *release.Metadata) string {
	return strings.Join([]string{
		fmt.Sprintf("%s=%d", ID, m.ID),
		fmt.Sprintf("%s=%s", Platform, m.Client),
		fmt.Sprintf("%s=%s", Version, m.Version),
		fmt.Sprintf("%s=%s", CreatedAt, m.CreatedAt.Format(time.RFC3339)),
		fmt.Sprintf("%s=%s", Signature, m.DSASignature),
		fmt.Sprintf("%s=%s", GitHash, m.GitHash),
		fmt.Sprintf("%s=%d", ReleasePercentage, m.ReleasePercentage),
	}, "\n")
}

// SerializeDeltas outputs a set of indexed, named Params corresponding to a list of Deltas
func SerializeDeltas(deltas ...*release.Delta) string {
	out := []string{
		fmt.Sprintf("%s=%d", NumDeltas, len(deltas)),
	}
	for i, d := range deltas {
		out = append(
			out,
			fmt.Sprintf("%s%s=%d", ID, deltaSuffix(i), d.ID),
			fmt.Sprintf("%s%s=%s", Platform, deltaSuffix(i), d.Client),
			fmt.Sprintf("%s%s=%s", FromVersion, deltaSuffix(i), d.FromVersion),
			fmt.Sprintf("%s%s=%s", ToVersion, deltaSuffix(i), d.ToVersion),
			fmt.Sprintf("%s%s=%s", Signature, deltaSuffix(i), d.DSASignature),
			fmt.Sprintf("%s%s=%s", CreatedAt, deltaSuffix(i), d.CreatedAt.Format(time.RFC3339)),
		)
	}
	return strings.Join(out, "\n")
}

// -

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

var cmdAdd = &cmd{
	name:    "add",
	argNum:  0,
	usage:   "release add",
	example: "release add",
	run: func(manager release.MetadataManager, args []string) {
		metadata, err := manager.Create(release.MetadataCreateArgs{
			Client:            GetParam(Platform).(release.Platform),
			Version:           GetParam(Version).(string),
			DSASignature:      GetParam(Signature).(string),
			GitHash:           GetParam(GitHash).(string),
			ReleasePercentage: GetParam(ReleasePercentage).(uint8),
		})
		if err != nil {
			log.Println("Error creating new release:", err.Error())
			os.Exit(1)
		}

		fmt.Println(Serialize(metadata))
	},
}

var cmdCurrentVersion = &cmd{
	name:   "currentversion",
	argNum: 1,
	usage:  "release currentversion",
	run: func(manager release.MetadataManager, args []string) {

		var platform release.Platform
		switch platformStr := args[0]; platformStr {
		case "windows":
			platform = release.Windows
		case "mac":
			platform = release.Mac
		case "linux":
			platform = release.Linux
		}

		// empty client id to get the latest release, not just the latest stable release
		metadata, err := manager.LatestCanary(platform)
		if err != nil {
			fmt.Println("Error getting latest release:", err.Error())
			os.Exit(3)
		}
		fmt.Println(metadata.Version)
	},
}

var cmdLatest = &cmd{
	name:   "latest",
	argNum: 0,
	usage:  "release latest",
	run: func(manager release.MetadataManager, args []string) {
		platform := GetParam(Platform).(release.Platform)

		// empty client id to get the latest release, not just the latest stable release
		metadata, err := manager.LatestCanary(platform)
		if err != nil {
			log.Println("Error getting latest release:", err.Error())
			os.Exit(1)
		}
		if metadata == nil {
			log.Println("No release", err.Error())
			os.Exit(1)
		}
		fmt.Println(Serialize(metadata))
	},
}

var cmdPublish = &cmd{
	name:   "publish",
	argNum: 3,
	usage:  "release publish <platform> <version> <release_percentage>",
	run: func(manager release.MetadataManager, args []string) {
		platformStr := args[0]
		versionStr := args[1]
		percentageStr := args[2]

		// get platform var (default to Mac because the type for these is unexported)
		platform := release.Mac
		switch platformStr {
		case "windows":
			platform = release.Windows
		case "linux":
			platform = release.Linux
		}

		// parse percentage value
		var canaryPercentage uint8
		if v, err := strconv.ParseUint(percentageStr, 10, 8); err != nil {
			fmt.Println("Error parsing canary percentage:", err.Error())
			os.Exit(4)
		} else {
			canaryPercentage = uint8(v)
		}

		if err := manager.Publish(platform, versionStr, canaryPercentage); err != nil {
			fmt.Println("Error setting canary percentage:", err.Error())
			os.Exit(4)
		}
		fmt.Printf("set canary percentage for %s to %d\n", args[0], canaryPercentage)
	},
}

// cmdAddDeltas adds a delta update to the release dd.
var cmdAddDeltas = &cmd{
	name:    "addDeltas",
	argNum:  0,
	usage:   "release addDeltas",
	example: "release addDeltas",
	run: func(manager release.MetadataManager, args []string) {
		var isErr bool
		var deltas []*release.Delta

		n := GetParam(NumDeltas).(int)
		for i := 0; i < n; i++ {
			platform := GetDeltaParam(i, Platform).(release.Platform)
			toVersion := GetDeltaParam(i, ToVersion).(string)
			fromVersion := GetDeltaParam(i, FromVersion).(string)
			signature := GetDeltaParam(i, Signature).(string)

			delta, err := manager.CreateDelta(platform, fromVersion, toVersion, signature)
			if err != nil {
				log.Println("Error creating new delta:", err.Error())
				isErr = true
				continue
			}
			deltas = append(deltas, delta)
		}

		fmt.Println(SerializeDeltas(deltas...))
		if isErr {
			os.Exit(1)
		}
	},
}

var latestDeltas = &cmd{
	name:   "latestDeltas",
	argNum: 0,
	usage:  "release latestDeltas",
	run: func(manager release.MetadataManager, args []string) {
		platform := GetParam(Platform).(release.Platform)
		version := GetParam(Version).(string)

		deltas, err := manager.DeltasToVersion(platform, version)
		if err != nil {
			log.Printf("Error getting deltas for release %s: %s", version, err.Error())
			os.Exit(1)
		}
		fmt.Println(SerializeDeltas(deltas...))
	},
}

// This is duplicated in services/release.kite.com/main.go for devops
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

var cmdMockServer = &cmd{
	name:        "mockserver",
	argNum:      0,
	usage:       "release mockserver",
	managerImpl: &release.MockMetadataManager{},
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

var cmdLocalServer = &cmd{
	name:    "localserver",
	argNum:  2,
	usage:   "release localserver <platform> <root>",
	example: "release localserver mac $GOPATH/src/github.com/kiteco/kiteco/osx",
	run: func(manager release.MetadataManager, args []string) {
		var platform release.Platform
		switch args[0] {
		case "mac":
			platform = release.Mac
		case "windows":
			platform = release.Windows
		case "linux":
			platform = release.Linux
		default:
			log.Fatalf("Invalid platform: %s", args[0])
		}
		router := mux.NewRouter()
		server := release.NewServer(router, manager, []release.Platform{platform})

		rootDir := args[1]
		if !filepath.IsAbs(rootDir) {
			log.Fatalf("Root directory path must be absolute: %s", rootDir)
		}
		server.SetReleaseRoot(rootDir)
		setHeaders := func(h http.Handler) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Set headers to download file
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Disposition", "attachment;")
				// Serve with the actual handler.
				h.ServeHTTP(w, r)
			}
		}
		prefix := "/" + release.LocalPrefix + "/"
		router.PathPrefix(prefix).Handler(http.StripPrefix(prefix,
			setHeaders(http.FileServer(http.Dir(rootDir)))))

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

	cmds := []*cmd{cmdAdd, cmdCurrentVersion, cmdLatest, cmdPublish, cmdAddDeltas, latestDeltas, cmdServer, cmdMockServer, cmdLocalServer}

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
