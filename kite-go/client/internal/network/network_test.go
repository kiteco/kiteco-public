package network

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ComponentInterfaces(t *testing.T) {
	m := NewManager(component.NewTestManager())
	component.TestImplements(t, m, component.Implements{
		Handlers:     true,
		Terminater:   true,
		Initializer:  true,
		KitedEventer: true,
	})
}

type MockTransport struct {
	succeed bool
	t       *testing.T
}

func (transport *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	assert.Equal(transport.t, req.URL.String(), "http://clients3.google.com/generate_204")
	if !transport.succeed {
		return nil, errors.New("FAIL")
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
		Header:     make(http.Header),
	}, nil
}

func Test_DoCheckOnline(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	require.NoError(t, err)
	defer s.Close()
	m := NewManager(s.Components)
	m.client.Transport = &MockTransport{false, t}
	m.doOnlineCheck(context.Background())
	assert.False(t, m.Online())

	m.client.Transport = &MockTransport{true, t}
	m.doOnlineCheck(context.Background())
	assert.True(t, m.Online())
}

//dummy implementer of NetworkEventer
type DummyEventer struct {
	Test bool
}

func (d *DummyEventer) NetworkOnline() {
	d.Test = true
}

func (d *DummyEventer) NetworkOffline() {
	d.Test = false
}

func (d *DummyEventer) KitedInitialized() {
	d.Test = true
}

func (d *DummyEventer) KitedUninitialized() {
	d.Test = false
}

func (d *DummyEventer) Name() string {
	return "dummy"
}

func Test_NetworkEventing(t *testing.T) {
	//setup componentMgr to call events
	componentMgr := component.NewTestManager()
	eventer := &DummyEventer{}
	err := componentMgr.Add(eventer)
	assert.NoError(t, err)
	componentMgr.NetworkOnline()
	assert.True(t, eventer.Test)
	componentMgr.NetworkOffline()
	assert.False(t, eventer.Test)
}

//test kited (un)initializing
func Test_KitedEventing(t *testing.T) {
	//setup componentMgr to call events
	componentMgr := component.NewTestManager()
	eventer := &DummyEventer{}
	err := componentMgr.Add(eventer)
	assert.NoError(t, err)
	componentMgr.KitedInitialized()
	assert.True(t, eventer.Test)
	componentMgr.KitedUninitialized()
	assert.False(t, eventer.Test)
}
