package localcode

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Client contains methods to interact with the Worker
type Client struct {
	workers   *workerGroup
	requests  *requestClient
	artifacts *artifactClient

	m        sync.Mutex
	contexts map[userMachine]*UserContext
}

// NewClient returns a Client to interact with the Worker
func NewClient() *Client {
	workers := newWorkerGroup()
	c := &Client{
		workers:   workers,
		requests:  newRequestClient(workers),
		artifacts: newArtifactClient(workers),
		contexts:  make(map[userMachine]*UserContext),
	}
	go c.loop()
	return c
}

// CreateUserContext returns a UserContext object for the provided user/machine. It will return
// an error if the context already exists
func (c *Client) CreateUserContext(uid int64, machine string) (*UserContext, error) {
	c.m.Lock()
	defer c.m.Unlock()

	um := userMachine{uid, machine}
	ctx, ok := c.contexts[um]
	if !ok {
		ctx = newUserContext(uid, machine, c.requests, c.artifacts)
		localfiles.Observe(uid, machine, ctx.observeFileSync)
		c.contexts[um] = ctx
		return ctx, nil
	}

	return nil, fmt.Errorf("localcode.Client: context already exists")
}

// UserContextOk returns a UserContext object for the provided user/machine. Will return
// false if there is no context for the provided user/machine.
func (c *Client) UserContextOk(uid int64, machine string) (*UserContext, bool) {
	c.m.Lock()
	defer c.m.Unlock()

	um := userMachine{uid, machine}
	ctx, ok := c.contexts[um]
	return ctx, ok
}

// Cleanup removes the provided context
func (c *Client) Cleanup(ctx *UserContext) {
	c.m.Lock()
	defer c.m.Unlock()

	localfiles.RemoveObserver(ctx.userID, ctx.machine)
	um := userMachine{ctx.userID, ctx.machine}
	delete(c.contexts, um)

	ctx.Cleanup()
}

// --

type byUserID []*UserContext

func (b byUserID) Len() int           { return len(b) }
func (b byUserID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byUserID) Less(i, j int) bool { return b[i].userID < b[j].userID }

// Handler is a simple plaintext endpoint to view information about current UserContext objects
func (c *Client) Handler(w http.ResponseWriter, r *http.Request) {
	c.m.Lock()
	defer c.m.Unlock()

	var contexts []*UserContext
	for _, context := range c.contexts {
		contexts = append(contexts, context)
	}

	sort.Sort(byUserID(contexts))

	tabw := tabwriter.NewWriter(w, 10, 10, 10, ' ', 0)
	defer tabw.Flush()

	tabw.Write([]byte("uid\tmachine\tmade_request\tpolling\tartifact\tage\tdirty\tfiles\terror\n"))
	for _, ctx := range contexts {
		func() {
			ctx.rw.Lock()
			defer ctx.rw.Unlock()
			var has bool
			for _, artifact := range ctx.artifacts {
				s := fmt.Sprintf("%d\t%s\t%t\t%d\t%s\t%s\t%t\t%d\t%s\n",
					ctx.userID, ctx.machine, ctx.madeRequest(), len(ctx.polling), artifact.UUID, time.Since(artifact.loadedAt),
					artifact.dirty(), len(artifact.IndexedPathHashes), artifact.Error)
				tabw.Write([]byte(s))
				has = true
			}
			for _, artifact := range ctx.errors {
				s := fmt.Sprintf("%d\t%s\t%t\t%d\t%s\t%s\t%t\t%d\t%s\n",
					ctx.userID, ctx.machine, ctx.madeRequest(), len(ctx.polling), artifact.UUID, time.Since(artifact.erroredAt),
					false, len(artifact.IndexedPathHashes), artifact.Error)
				tabw.Write([]byte(s))
				has = true
			}
			if !has {
				s := fmt.Sprintf("%d\t%s\t%t\t%d\t%s\t%s\t%t\t%d\t%s\n",
					ctx.userID, ctx.machine, ctx.madeRequest(), len(ctx.polling), "", "", false, 0, "")
				tabw.Write([]byte(s))
			}
		}()
	}
}

// --

var (
	cleanupInterval     = time.Minute
	maxInactiveArtifact = time.Minute * 15
)

func (c *Client) loop() {
	cleanupTicker := time.Tick(cleanupInterval)

	for {
		select {
		case <-cleanupTicker:
			c.cleanup()
		}
	}
}

func (c *Client) cleanup() {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	defer cleanupDuration.DeferRecord(time.Now())

	c.m.Lock()
	defer c.m.Unlock()

	var hasIndex int64
	var hadIndex int64
	var requestedIndex int64
	var hasErroredIndex int64
	var hasRequestedIndex int64
	var hasRequestedIndexDirty int64
	for _, ctx := range c.contexts {
		ctx.cleanupInactive()
		if ctx.hasArtifact() {
			hasIndex++
		}
		if ctx.madeRequest() {
			requestedIndex++
			if ctx.gotArtifact() {
				hadIndex++
			}
			if ctx.hasArtifact() {
				hasRequestedIndex++
				if ctx.hasDirtyArtifact() {
					hasRequestedIndexDirty++
				}
			} else if ctx.hasErroredArtifact() {
				hasErroredIndex++
			}
		}
	}

	haveIndexRatio.Set(hasIndex, int64(len(c.contexts)))
	hadIndexWithRequestRatio.Set(hadIndex, requestedIndex)
	haveIndexWithRequestRatio.Set(hasRequestedIndex, requestedIndex)
	haveErrorWithRequestRatio.Set(hasErroredIndex, requestedIndex)
	haveDirtyIndexWithRequestRatio.Set(hasRequestedIndexDirty, hasRequestedIndex)
}

// --

type userMachine struct {
	UserID  int64
	Machine string
}

type userMachineFile struct {
	UserID   int64
	Machine  string
	Filename string
}
