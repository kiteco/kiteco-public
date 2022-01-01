package platform

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgryski/go-spooky"

	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/platform/installid"
	"github.com/kiteco/kiteco/kite-go/client/platform/machine"
	"github.com/kiteco/kiteco/kite-go/client/platform/messagebox"
	"github.com/kiteco/kiteco/kite-go/client/platform/version"
)

const (
	logFlags                = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	maxLogFiles             = 15
	ggnnSubtokenEnabledFlag = "GGNN_SUBTOKEN"
)

var (
	// LogPrefix is the prefix string for client logs
	LogPrefix = fmt.Sprintf("[%s] ", "kited")
)

// Platform encapsulates platform specific code and is used in the client app
type Platform struct {
	// Logger is configured with Kite's patterns and writes into Kite's client.log file
	Logger *log.Logger
	// LogWriter writes into the client.log file
	LogWriter io.Writer
	// KiteRoot is the root path to Kite's user data directory
	KiteRoot string
	// LogDir is the root path to Kite's directory which contains the log files
	LogDir string
	// Version is the version of the currently executed instance of Kite
	ClientVersion string
	// MachineID is a unique ID of the local machine, it's static across restarts and reinstalls on the same machine
	MachineID string
	// InstallID is a UUID, it's persisted across restarts. It's not static across reinstalls on the same machine or on other machines of the same user.
	InstallID string
	// IsDevMode is true if the debug options (e.g. the servers menu) should be displayed
	// it's true if the binary is a debug build or if $KITEROOT/DEBUG exists
	DevMode bool
	// IsDebugBuild is true if the current build was build on a developers machine
	IsDebugBuild bool
	// IsUnitTestMode is true if the client is run in a unit test
	IsUnitTestMode bool
	// GGNNSubtokenEnabled indicates whether GGNN subtoken partial completions are enabled
	GGNNSubtokenEnabled bool
	// GGNNSubtokenEnabledByFlag indicates that GGNN Subtoken is enabled with a flag present in the filesystem
	GGNNSubtokenEnabledByFlag bool
	// path to the client.log file
	LogFile string
	// set if this installation was the first time Kite was run on this machine
	IsNewInstall bool
	// testFeatureOverride allows to disable flags for certain test cases which are incompatible with env var overrides
	testFeatureOverride map[string]bool
}

// NewPlatform creates a new instance of the Platform, it returns a fully initialized platform instance
// If an empty root dir is passed then the directory $HOME/.kite will be used
func NewPlatform() (*Platform, error) {
	p := newPlatform(kiteRoot(), true)
	if err := p.Initialize(); err != nil {
		return nil, err
	}
	return p, nil
}

// NewTestPlatform creates a new instance of the Platform, to be used in unit tests.
// if rootDir is empty, then a temp directory will be used
func NewTestPlatform(rootDir string) (*Platform, error) {
	return NewTestPlatformFeatures(rootDir, nil)
}

// NewTestPlatformFeatures creates a new instance of the Platform, to be used in unit tests.
// if rootDir is empty, then a temp directory will be used
// featureOverride allows to customize flags
func NewTestPlatformFeatures(rootDir string, featureOverride map[string]bool) (*Platform, error) {
	if rootDir == "" {
		root, err := ioutil.TempDir("", "kite-test")
		if err != nil {
			return nil, err
		}

		rootDir = root
	}

	p := newPlatform(rootDir, false)
	p.IsUnitTestMode = true
	p.testFeatureOverride = featureOverride
	if err := p.Initialize(); err != nil {
		return nil, err
	}

	return p, nil
}

func newPlatform(rootDir string, newInstallEvent bool) *Platform {
	// if the root directory does not exist, this is most likely a new install...
	var newInstall bool
	_, err := os.Stat(rootDir)
	if err != nil && os.IsNotExist(err) {
		newInstall = true
	}

	if newInstall && newInstallEvent {
		clienttelemetry.Event("New Install", nil)
	}

	return &Platform{
		KiteRoot:     rootDir,
		LogDir:       filepath.Join(rootDir, "logs"),
		IsNewInstall: newInstall,
	}
}

// Initialize sets up the
func (p *Platform) Initialize() error {
	if p.IsNewInstall {
		_ = os.MkdirAll(p.KiteRoot, 0700)
	}

	// DebugBuild && DevMode checks,
	p.IsDebugBuild = version.IsDebugBuild()
	p.DevMode = version.IsDevMode() || p.IsDebugBuild
	if !p.DevMode {
		p.DevMode = p.IsFeatureEnabled("DEBUG")
	}

	// client version
	p.ClientVersion = version.Version()

	// setup machine id
	var err error
	p.MachineID, err = machine.ID(p.DevMode)
	if err != nil {
		return err
	}

	// setup install id
	p.InstallID, err = installid.LoadInstallID(p.KiteRoot)
	if err != nil {
		return err
	}

	// TODO(naman) get rid of GGNN Subtoken
	if p.IsFeatureEnabled(ggnnSubtokenEnabledFlag) {
		p.GGNNSubtokenEnabled = true
		p.GGNNSubtokenEnabledByFlag = true
	} else {
		// If there's no flag file present for subtoken decoding,
		// we activate it for 25 % of the users
		// based on their install id so we can ensure deterministic results
		h := spooky.Hash64([]byte(p.InstallID))
		seed := int64((^uint64(1 << 63)) & h) // take the lower 63 bits of the hash

		if rand.New(rand.NewSource(seed)).Float64() < .25 {
			p.GGNNSubtokenEnabled = true
		}
	}

	return nil
}

// InitializeLogger initializes and rotates logs
func (p *Platform) InitializeLogger() error {
	// setup log file
	err := os.MkdirAll(p.LogDir, os.ModePerm)
	if err != nil {
		return err
	}
	p.LogFile = filepath.Join(p.LogDir, "client.log")

	// rotate before the logwriter is created
	rotateLogs(p.LogFile, maxLogFiles)

	// get the platform-specific log writer
	// enable additional output to stdout in unit tests
	logwr, err := logWriter(p.LogFile, p.IsUnitTestMode)
	if err != nil {
		return err
	}

	p.LogWriter = logwr

	// configure global logger
	log.SetPrefix(LogPrefix)
	log.SetFlags(logFlags)
	if !p.IsUnitTestMode {
		log.SetOutput(logwr)
	}

	p.Logger = log.New(logwr, LogPrefix, logFlags)
	return nil
}

// IsFeatureEnabled returns true if $kiteRoot/$name exists in the filesystem
func (p *Platform) IsFeatureEnabled(name string) bool {
	// check override before env variables
	if p.testFeatureOverride != nil {
		v, ok := p.testFeatureOverride[name]
		if ok {
			return v
		}
	}

	// allow to enable a feature by setting an environment variable KITE_$name (with name in uppercase letters)
	if p.IsUnitTestMode && os.Getenv(fmt.Sprintf("KITE_%s", strings.ToUpper(name))) != "" {
		return true
	}

	path := p.featureFlagPath(name)
	if _, err := os.Stat(path); err == nil || os.IsExist(err) {
		return true
	}
	return false
}

func (p *Platform) featureFlagPath(name string) string {
	return filepath.Join(p.KiteRoot, name)
}

// newLogger creates a new logger which writes into the global logfile
func newLogger(logwriter io.Writer) *log.Logger {
	return log.New(logwriter, LogPrefix, logFlags)
}

// ShowAlert displays an error message box with the given title and message
func ShowAlert(message string) {
	opts := messagebox.Options{
		Title: "Kite",
		Text:  message,
	}
	messagebox.ShowAlert(opts)
}

// DispatchWarning displays a warning message box with the given key, title and message.
// The key controls warning suppression on OS X.
// On OS X, it should not be called by the main thread (inside initialization routines).
func DispatchWarning(key, message, info string) {
	opts := messagebox.Options{
		Key:   key,
		Title: "Kite",
		Text:  message,
		Info:  info,
	}
	if err := messagebox.DispatchWarning(opts); err != nil {
		log.Println("failed to show warning", err)
	}
}
