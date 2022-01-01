package component

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests that components with an empty name can't be added
func TestEmpyComponentName(t *testing.T) {
	c := &CountingComponent{name: ""}

	m := NewTestManager()
	err := m.Add(c)

	assert.Error(t, err)
}

// Tests that components with an empty name can't be added
func TestAlreadyRegistered(t *testing.T) {
	c1 := &CountingComponent{name: "test"}
	c2 := &CountingComponent{name: "test"}

	m := NewTestManager()
	err := m.Add(c1)
	assert.NoError(t, err)

	err = m.Add(c2)
	assert.Error(t, err)
}

type recursiveComponent struct {
	manager *Manager
	called  map[string]bool
}

func newRecursiveComponent(manager *Manager) *recursiveComponent {
	return &recursiveComponent{
		manager: manager,
		called:  make(map[string]bool),
	}
}

func (r *recursiveComponent) Name() string {
	return "recursive-test"
}

func (r *recursiveComponent) Initialize(opts InitializerOptions) {
	r.called["initialize"] = true
	r.manager.LoggedIn()
}

func (r *recursiveComponent) LoggedIn() {
	r.called["loggedin"] = true
	r.manager.LoggedOut()
}

func (r *recursiveComponent) LoggedOut() {
	r.called["loggedout"] = true
}

func TestDeadlock(t *testing.T) {
	m := NewTestManager()
	r := newRecursiveComponent(m)
	TestImplements(t, r, Implements{
		Initializer: true,
		UserAuth:    true,
	})

	m.Add(r)
	m.Initialize(InitializerOptions{})
	require.Contains(t, r.called, "initialize")
	require.Contains(t, r.called, "loggedin")
	require.Contains(t, r.called, "loggedout")
}

// Tests the propagation of method calls to all registered components
func TestWorkflow(t *testing.T) {
	c := NewCountingComponent("counter")

	m := NewTestManager()
	m.Add(c)

	m.RegisterHandlers(nil)
	assert.EqualValues(t, 1, c.GetRegisterHandlersCount())

	m.Initialize(InitializerOptions{})
	assert.EqualValues(t, 1, c.GetInitCount())

	//login & logout
	m.LoggedIn()
	assert.EqualValues(t, 1, c.GetLoggedInCount())
	m.LoggedOut()
	assert.EqualValues(t, 1, c.GetLoggedOutCount())

	//login again
	m.LoggedIn()
	assert.EqualValues(t, 2, c.GetLoggedInCount())

	//events
	m.PluginEvent(nil)
	assert.EqualValues(t, 1, c.GetPluginEventCount())

	m.EventResponse(nil)
	assert.EqualValues(t, 1, c.GetEventResponseCount())

	m.ProcessedEvent(nil, nil)
	assert.EqualValues(t, 1, c.GetPluginEventCount())

	//settings
	m.Deleted("key1")
	assert.EqualValues(t, 1, c.GetSettingsDeletedCount())

	m.Updated("key1", "value1")
	assert.EqualValues(t, 1, c.GetSettingsUpdatedCount())

	//terminate
	m.Terminate()
	assert.EqualValues(t, 1, c.GetTerminateCount())

	//reset
	c.Reset()
	assert.EqualValues(t, 0, c.GetTerminateCount())
}

//make sure that GoTick starts goroutines for each component
func Test_GoTick(t *testing.T) {
	componentDelay := time.Second / 2
	ctxTimeout := 750 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(3)

	m := NewTestManager()
	m.Add(&delayingComponent{name: "component 1", delay: componentDelay, onFinish: wg.Done})
	m.Add(&delayingComponent{name: "component 2", delay: componentDelay, onFinish: wg.Done})
	m.Add(&delayingComponent{name: "component 3", delay: componentDelay, onFinish: wg.Done})

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	m.GoTick(ctx)

	goTickDuration := time.Since(start)
	assert.True(t, goTickDuration.Nanoseconds() <= (100*time.Millisecond).Nanoseconds(), "component managers goTick() must return immediately. Duration: %s", goTickDuration.String())

	//wait until all components finished GoTick()
	wg.Wait()
	overall := time.Now().Sub(start)
	assert.True(t, overall.Nanoseconds() >= componentDelay.Nanoseconds(), "Overall processing time must be at least the time each component needs")
	assert.False(t, overall.Nanoseconds() > ctxTimeout.Nanoseconds(), "Overall processing time must not be longer than the context timeout")
}

// dummy component which needs a fixed duration to finish GoTick()

type delayingComponent struct {
	name     string
	delay    time.Duration
	onFinish func()
}

func (d *delayingComponent) Name() string {
	return d.name
}

func (d *delayingComponent) GoTick(ctx context.Context) {
	time.Sleep(d.delay)
	d.onFinish()
}
