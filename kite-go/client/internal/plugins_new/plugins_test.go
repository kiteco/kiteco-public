package plugins

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	m := NewManager(system.Options{})
	component.TestImplements(t, m, component.Implements{
		Initializer:      true,
		Handlers:         true,
		ProcessedEventer: true,
		Ticker:           true,
	})
}

func Test_GetEmptyEncounteredEditors(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = s.SetupComponents(nil, settings.NewTestManager(), nil, nil, NewManager(system.Options{DevMode: false}))
	assert.NoError(t, err)

	resp, err := s.DoKitedGet("/clientapi/plugins/encountered")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.NotNil(t, resp.Body)
	buf, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var enc map[string]bool
	err = json.Unmarshal(buf, &enc)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(enc))
}

func Test_GetNonEmptyEncounteredEditors(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = s.SetupComponents(nil, settings.NewTestManager(), nil, nil, NewManager(system.Options{DevMode: false}))
	assert.NoError(t, err)

	toAdd := []string{"atom", "vscode"}
	body, err := json.Marshal(toAdd)
	assert.NoError(t, err)

	resp, err := s.DoKitedPost("/clientapi/plugins/encountered", strings.NewReader(string(body)))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.NotNil(t, resp.Body)
	buf, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var enc map[string]bool
	err = json.Unmarshal(buf, &enc)
	assert.NoError(t, err)

	assert.Equal(t, len(toAdd), len(enc))
	for _, e := range toAdd {
		assert.True(t, enc[e])
	}

	resp, err = s.DoKitedGet("/clientapi/plugins/encountered")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.NotNil(t, resp.Body)
	buf, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	err = json.Unmarshal(buf, &enc)
	assert.NoError(t, err)

	assert.Equal(t, len(toAdd), len(enc))
	for _, e := range toAdd {
		assert.True(t, enc[e])
	}
}

type withTraits interface {
	Traits() map[string]interface{}
}

func Test_OnboardingTracking(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	// the kitelocal component handles the event processing
	kiteLocalMgr, err := kitelocal.NewManager(s.Components, kitelocal.Options{
		IndexedDir: s.BasePath,
	})
	require.NoError(t, err)

	pluginMgr := NewManager(system.Options{DevMode: true})
	pluginMgr.onboardingDir = s.BasePath

	err = s.SetupComponents(nil,
		settings.NewTestManager(),
		permissions.NewManager([]lang.Language{lang.Python}, nil),
		metrics.NewMockManager(),
		pluginMgr, kiteLocalMgr)
	require.NoError(t, err)

	var filepath string
	err = s.KitedClient.GetJSON("/clientapi/plugins/onboarding_file?editor=atom", &filepath)
	require.NoError(t, err)
	defer os.Remove(filepath)

	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	_, err = s.KitedClient.PostEventData("atom", filepath, string(content), "edit", int64(float32(len(content))*0.7), int64(float32(len(content))*0.7), stringindex.UTF8)
	require.NoError(t, err)

	uids := userids.NewUserIDs("install-id", "machine-id")
	uids.SetUser(42, "user@example.com", true)
	clienttelemetry.SetUserIDs(uids)

	traits, ok := s.Metrics.(withTraits)
	require.True(t, ok, "expected a mock metrics manager")

	for i := 0; i < 40; i++ {
		if len(traits.Traits()) != 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	userTraits := traits.Traits()
	require.Empty(t, userTraits, "onboarding must not be completed with an offset at 80%, the minimum is 90%")

	// atom event
	_, err = s.KitedClient.PostEventData("atom", filepath, string(content), "edit", int64(len(content)), int64(len(content)), stringindex.UTF8)
	require.NoError(t, err)
	waitForTraits(traits, 1)
	userTraits = traits.Traits()
	require.Len(t, userTraits, 1)
	require.EqualValues(t, true, userTraits["atom_onboarding_completed"])

	// pycharm event
	_, err = s.KitedClient.PostEventData("pycharm", filepath, string(content), "edit", int64(len(content)), int64(len(content)), stringindex.UTF8)
	require.NoError(t, err)
	waitForTraits(traits, 2)
	userTraits = traits.Traits()
	require.Len(t, userTraits, 2)
	require.EqualValues(t, true, userTraits["atom_onboarding_completed"])
	require.EqualValues(t, true, userTraits["pycharm_onboarding_completed"])
}

func Test_OnboardingFilesPython(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	pluginMgr := NewManager(system.Options{DevMode: true})
	pluginMgr.onboardingDir = s.BasePath

	err = s.SetupComponents(nil,
		settings.NewTestManager(),
		permissions.NewManager([]lang.Language{lang.Python}, nil),
		metrics.NewMockManager(),
		pluginMgr)
	require.NoError(t, err)

	var filepath string
	defer os.Remove(filepath)

	// editors with assets per os and editor
	for _, editor := range []string{"atom", "intellij", "vscode"} {
		err = s.KitedClient.GetJSON("/clientapi/plugins/onboarding_file?editor="+editor, &filepath)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath)
		require.NoError(t, err)
		require.Regexp(t, ".+\\.py$", filepath)

		assetData, err := Asset(fmt.Sprintf("onboarding/%s/kite_tutorial_%s_%s.py", editor, editor, runtime.GOOS))
		require.NoError(t, err)
		assert.EqualValues(t, assetData, data)
	}

	// editors with assets per editor
	for _, editor := range []string{"spyder", "sublime3", "vim"} {
		err = s.KitedClient.GetJSON("/clientapi/plugins/onboarding_file?editor="+editor, &filepath)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath)
		require.NoError(t, err)
		require.Regexp(t, ".+\\.py$", filepath)

		assetData, err := Asset(fmt.Sprintf("onboarding/%s/kite_tutorial_%s.py", editor, editor))
		require.NoError(t, err)
		assert.EqualValues(t, assetData, data)
	}
}

func Test_OnboardingFilesGolang(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	pluginMgr := NewManager(system.Options{DevMode: true})
	pluginMgr.onboardingDir = s.BasePath

	err = s.SetupComponents(nil,
		settings.NewTestManager(),
		permissions.NewManager([]lang.Language{lang.Python}, nil),
		metrics.NewMockManager(),
		pluginMgr)
	require.NoError(t, err)

	var filepath string
	defer os.Remove(filepath)

	// editors with assets per editor
	for _, editor := range []string{"intellij"} {
		err = s.KitedClient.GetJSON("/clientapi/plugins/onboarding_file?language=go&editor="+editor, &filepath)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath)
		require.NoError(t, err)
		require.Regexp(t, ".+\\.go$", filepath, "filepath must be a .go file")

		assetData, err := Asset(fmt.Sprintf("onboarding/%s/kite_tutorial_%s.golang", editor, editor))
		require.NoError(t, err)
		assert.EqualValues(t, assetData, data)
	}
}

func waitForTraits(traits withTraits, expected int) {
	for i := 0; i < 40; i++ {
		if len(traits.Traits()) == expected {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
}
