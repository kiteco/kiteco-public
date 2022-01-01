package client

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/startup"
)

const (
	defaultHTTPTimeout = 10 * time.Second

	// tickInterval must be >= tickTimeout
	componentTickInterval = 15 * time.Second
	componentTickTimeout  = 10 * time.Second

	authRetryInterval = 10 * time.Second

	uploadLogsOnStart = false
)

var (
	componentTicker = time.Tick(componentTickInterval)
	authRetryTicker = time.Tick(authRetryInterval)
)

func (c *Client) processHTTP(ctx context.Context, target string) error {
	log.Printf("starting processHTTP(%s)", target)

	clienttelemetry.SetUserIDs(c.UserIDs)

	defer func() {
		log.Printf("stopping processHTTP(%s)", target)

		c.mu.Lock()
		defer c.mu.Unlock()
		if c.cancelFunc != nil {
			c.cancelFunc = nil
		}
	}()

	targetURL, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("error parsing target url %s: %s", target, err)
	}

	// Set destination for proxied URLs
	c.AuthClient.SetTarget(targetURL)
	defer c.AuthClient.UnsetTarget()

	// Check for updates
	c.Updater.CheckForUpdates(false)

	hasCookie := c.AuthClient.HasAuthCookie()
	remoteUser, remoteErr := c.AuthClient.FetchUser(ctx)
	localUser, localErr := c.AuthClient.CachedUser()
	switch {
	// If we were able to authenticate remotely, log in. This means the user has a valid
	// session and we were able to fetch the user object remotely
	case remoteErr == nil:
		c.AuthClient.LoggedInChan() <- remoteUser
		// User will identify in the select loop below via normal login flow

	// If we have an auth cookie and a cached user, treat the user as logged in. This means
	// the user had a valid session before, and a user object was cached during that session.
	// But we currently cannot fetch the user remotely (i.e user is offline)
	case hasCookie && localErr == nil:
		c.AuthClient.LoggedInChan() <- localUser
		// User will identify in the select loop below via normal login flow

	// If we don't have an auth cookie, aren't logged in. This means we don't see a valid
	// session, but we have a cached user. We should use the userID in metrics, but NOT log in
	case !hasCookie && localErr == nil:
		c.UserIDs.SetUser(localUser.ID, localUser.Email, false)

		// Metrics will identify with user id
		c.Metrics.Identify()

	// User was not authenticated, and cached user not found. So register the installid if it has not been registered.
	default:
		channel := startup.GetChannel(os.Args)
		if created, ok := c.Settings.GetBool("iid_registered"); !ok || !created {
			props := map[string]interface{}{
				"channel": channel,
			}
			clienttelemetry.InstallIDs.Event("kite_install_started", props)
			clienttelemetry.InstallIDs.Update(props)
			c.Settings.Set("iid_registered", "true")
			c.Settings.Set(settings.InstallTimeKey, time.Now().Format(time.RFC3339))
		} else {
			clienttelemetry.InstallIDs.Update(map[string]interface{}{
				"channel": channel,
			})
		}

		// Metrics will identify with install id
		c.Metrics.Identify()
	}

	// This is used to keep track of whether or not the client has finished initializing
	c.components.KitedInitialized()
	defer func() {
		c.components.KitedUninitialized()
	}()

	// Set the user/plan to nil when this method returns so that it is correctly reset
	// when the method restarts (vs using an older version). Note, actually logging out
	// here may be too aggressive, as errors/panics in this code that trigger this defer
	// probably shouldn't log people out.
	defer c.AuthClient.SetUser(nil)

	if !c.logsUploaded && uploadLogsOnStart {
		c.logsUploaded = true
		go uploadLogs(c.AuthClient, c.Platform.LogDir, c.Platform.MachineID, c.Platform.InstallID)
	}

	// This is used to notify tests that the client has finished initializing
	// NOTE: This signal should always be as close as possible to the beginning of
	// the main processing loop. This signals to our tests that the client is ready
	// to process events. The further this is from the actual loop, the more likely
	// the test is to hit a race where all events are fired before this loop can
	// process events, causing dropped events, and counters not matching in tests.
	atomic.StoreInt32(&c.testReady, 1)
	defer func() {
		atomic.StoreInt32(&c.testReady, 0)
	}()

	// Fire initial tick
	c.componentGoTick(ctx)

	var loggedIn bool
	for {
		select {
		case <-ctx.Done():
			log.Println("got disconnect message")
			return nil

		case user := <-c.AuthClient.LoggedInChan():
			log.Println("received login event")
			c.UserIDs.SetUser(user.ID, user.Email, true)
			c.AuthClient.SetUser(user)
			c.Metrics.Identify()
			c.components.LoggedIn()
			loggedIn = true

		case <-c.AuthClient.LoggedOutChan():
			log.Println("received logout event")
			c.UserIDs.Logout()
			c.AuthClient.SetUser(nil)
			if loggedIn {
				// only notify of logout if a user was logged in
				c.components.LoggedOut()
				loggedIn = false
			}

		case <-componentTicker:
			c.componentGoTick(ctx)

		//fixme(jansorg): Is this a no-op?
		case <-c.Updater.ReadyChan():
			// an update is ready to install, notify the sidebar
			log.Printf("an update is ready")

		case r := <-c.kitelocal.Responses:
			// Notify completions.Manager and all other components about this response
			c.components.EventResponse(r)
		}
	}
}

// componentGoTick is a helper method to fire component.GoTick
func (c *Client) componentGoTick(ctx context.Context) {
	go func() {
		tickCtx, tickCancel := context.WithTimeout(ctx, componentTickTimeout)
		defer tickCancel()

		c.components.GoTick(ctx)
		select {
		case <-tickCtx.Done():
			return
		}
	}()
}
