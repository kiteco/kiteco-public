package clientapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
	"github.com/pkg/errors"
)

// StartEmptyTestEnvironment creates an empty project and starts the kited client
func StartEmptyTestEnvironment(components ...component.Core) (*TestEnvironment, error) {
	p, err := NewTestEnvironment()
	if err != nil {
		return nil, err
	}

	err = p.StartPortNoDists(0, components...)
	return p, err
}

// StartDefaultTestEnvironment creates a project with preconfigured files and start the kited client.
func StartDefaultTestEnvironment(loginUser bool, clientOpts *client.Options, components ...component.Core) (*TestEnvironment, error) {
	p, err := NewTestEnvironment()
	if err != nil {
		return nil, err
	}

	setupDefaultFiles(p)

	// if using default opts, then preload the builtin distribution to avoid flaky tests
	if clientOpts == nil {
		clientOpts = &client.Options{
			LocalOpts: kitelocal.Options{
				Dists: []keytypes.Distribution{
					keytypes.BuiltinDistribution3,
				},
			},
		}
	}

	err = p.StartPort(0, clientOpts, components...)
	if err != nil {
		return p, fmt.Errorf("client startup failed with error: %s", err.Error())
	}

	if loginUser {
		resp, err := p.KitedClient.SendLoginRequest("user@example.com", "secret", true)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("SendLoginRequest failed with error: %s", err.Error())
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("SendLoginRequest failed with unexpected status: %d/%s", resp.StatusCode, resp.Status)
		}
	}

	return p, err
}

// returns a set of whitelisted, blacklisted and ignored files
func setupDefaultFiles(p *TestEnvironment) {
	// setup 5 sample files
	for i := 0; i < 5; i++ {
		f := filepath.Join(p.DataDirPath, fmt.Sprintf("file_%d.py", i))
		ioutil.WriteFile(f, []byte("import json"), 0600)
		p.Files = append(p.Files, f)
	}
}

// NewTestEnvironment creates a new, empty project with the given feature flags enabled in the underlying platform
func NewTestEnvironment() (*TestEnvironment, error) {
	// Use homedir because the default temp directories are filtered by Kite on some platforms
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dataDir, err := ioutil.TempDir(usr.HomeDir, "kite-user-data")
	if err != nil {
		return nil, err
	}

	backend, err := mockserver.NewBackend(map[string]string{"user@example.com": "secret", "pro@example.com": "secret"})
	if err != nil {
		return nil, fmt.Errorf("NewBackend failed with error: %s", err.Error())
	}
	backend.SetUserPlan("pro@example.com", true)

	ctx, cancel := context.WithCancel(context.Background())
	return &TestEnvironment{
		ctx:         ctx,
		ctxCancel:   cancel,
		DataDirPath: dataDir,
		Backend:     backend,
	}, nil
}

// TestEnvironment is a test setup which uses the real kited server
type TestEnvironment struct {
	ctx         context.Context
	ctxCancel   func()
	Server      *http.Server
	DataDirPath string
	Backend     *mockserver.MockBackendServer
	Kited       *client.Client
	KitedClient *mockserver.KitedClient
	MockTracker *telemetry.MockClient
	MockUpdater *updates.MockManager
	Files       []string
}

// StartPortNoDists activates the project without any Python dists loaded by kitelocal
func (p *TestEnvironment) StartPortNoDists(port int, components ...component.Core) error {
	return p.StartPort(port, &client.Options{
		LicenseStore: licensing.NewStore(p.Backend.Authority().CreateValidator(), ""),
		LocalOpts: kitelocal.Options{
			Dists: []keytypes.Distribution{},
		},
	}, components...)
}

// StartPort activates the project and stars kited on the given port. Use '0' to let kited choose its own port.
func (p *TestEnvironment) StartPort(port int, customOpts *client.Options, components ...component.Core) error {
	var opts client.Options
	if customOpts != nil {
		opts = *customOpts
	}

	if opts.TestRootDir == "" {
		opts.TestRootDir = p.DataDirPath
	}
	if opts.LocalOpts.IndexedDir == "" {
		opts.LocalOpts.IndexedDir = opts.TestRootDir
	}
	if opts.LicenseStore == nil {
		opts.LicenseStore = licensing.NewStore(p.Backend.Authority().CreateValidator(), "")
	}

	kited, server, mockTracker, err := StartTestClient(p.ctx, port, &opts, components...)
	if err != nil {
		return errors.Wrap(err, "client setup failed")
	}

	// run client main loop in background
	go func() {
		if err := kited.Connect(p.Backend.URL.String()); err != nil {
			log.Fatalf("connect failed: %s", err.Error())
		}
	}()

	kitedClient := mockserver.NewKitedClient(kited.URL)

	p.Kited = kited
	p.KitedClient = kitedClient
	p.Server = server
	p.MockTracker = mockTracker

	switch m := p.Kited.Updater.(type) {
	case *updates.MockManager:
		p.MockUpdater = m
	default:
		return errors.New("mock updater not accessible")
	}

	// Wait for client to be initialized before returning
	return p.WaitForReady(10 * time.Second)
}

// WaitForReady waits for the client in the test environment to initialize
func (p *TestEnvironment) WaitForReady(timeout time.Duration) error {
	// Try to wait for a bit so that the client has time to start
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if p.Kited.TestReady() {
				return nil
			}
		}
	}
}

// WaitForNotReady waits for the client in the test environment to disconnect
func (p *TestEnvironment) WaitForNotReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if !p.Kited.TestReady() {
				return nil
			}
		}
	}
}

// Close releases resources
func (p *TestEnvironment) Close() {
	p.ctxCancel()

	// shutdown kited's HTTP server
	if p.Server != nil {
		if err := p.Server.Close(); err != nil {
			log.Printf("error shuttding down kited HTTP server: %s", err.Error())
		}
	}

	// disconnect kited from backend
	if p.Kited != nil {
		p.Kited.Shutdown()
	}

	// mock backend
	if p.Backend != nil {
		p.Backend.Close()
	}

	clienttelemetry.Close()

	os.RemoveAll(p.DataDirPath)
}

// TestFlush calls TestFlush on the component manager of kited
func (p *TestEnvironment) TestFlush(ctx context.Context) {
	p.Kited.TestComponentManager().TestFlush(ctx)
}

// SetOffline calls SetOffline on the network manager of kited
func (p *TestEnvironment) SetOffline(offline bool) {
	p.Kited.Network.SetOffline(true)
}
