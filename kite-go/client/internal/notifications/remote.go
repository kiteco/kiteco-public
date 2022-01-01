package notifications

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

func (m *Manager) setNotifForID(id, html string) error {
	ndir := m.notifdir.Load()
	if ndir == nil {
		return errors.Errorf("component has not been initialized")
	}

	if _, ok := conversion.NotifIDSet[id]; !ok {
		return errors.Errorf("%s is an unknown notification id", id)
	}

	fp := filepath.Join(ndir.(string), fmt.Sprintf("%s.html", id))
	if err := ioutil.WriteFile(fp, []byte(html), os.ModePerm); err != nil {
		return errors.Errorf("error writing notification to file: %v", err)
	}
	return nil
}

func (m *Manager) staticID(id string) string {
	if id == conversion.CompletionsCTAPostTrial {
		return ProCompletionCTA(m.uid())
	}
	return id
}

type payload struct {
	HTML string `json:"html"`
	ID   string `json:"id,omitempty"`
}

// HandleRemoteMessage handles remote control messages
func (m *Manager) HandleRemoteMessage(msg remotectrl.Message) error {
	switch msg.Type {
	case remotectrl.DesktopNotification:
	case remotectrl.SetNotification:
	default:
		return nil
	}

	var pl payload
	if err := json.Unmarshal(msg.Payload, &pl); err != nil {
		rollbar.Error(errors.New("could not unmarshal remote message payload"), msg.Payload, err.Error())
		return err
	}
	switch msg.Type {
	case remotectrl.DesktopNotification:
		err := m.showPayload(pl.HTML)
		if err != nil {
			rollbar.Error(errors.New("Failed to show desktop_notification"), err.Error())
		}
		return err
	case remotectrl.SetNotification:
		err := m.setNotifForID(pl.ID, pl.HTML)
		if err != nil {
			rollbar.Error(errors.New("Failed to set_notification"), err.Error())
		}
		return err
	}
	return nil
}

// ShowNotificationByID shows the notification associated with an id.
func (m *Manager) ShowNotificationByID(id string) error {
	if _, ok := conversion.NotifIDSet[id]; !ok {
		return errors.Errorf("%s is an unknown notification id", id)
	}
	return m.notify(id)
}
