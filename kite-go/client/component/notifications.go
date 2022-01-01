package component

import (
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
)

// NotificationsManager ...
type NotificationsManager interface {
	remotectrl.Handler
	ShowNotificationByID(id string) error
}

// MockNotify implements component.NotificationsManager
// To allow spying on the function calls
type MockNotify struct {
	id    string
	rcMsg *remotectrl.Message
}

// ShowNotificationByID implements component.NotificationsManager
func (mn *MockNotify) ShowNotificationByID(id string) error {
	mn.id = id
	return nil
}

// ShowNotifCalledWith returns what ShowNotificationByID was called with
func (mn *MockNotify) ShowNotifCalledWith() string {
	return mn.id
}

// ShowNotifCalled returns whether ShowNotificationByID was called
func (mn *MockNotify) ShowNotifCalled() bool {
	return mn.id != ""
}

// HandleRemoteMessage implements remotectrl.Message
func (mn *MockNotify) HandleRemoteMessage(msg remotectrl.Message) error {
	mn.rcMsg = &msg
	return nil
}

// HandleRCCalledWith returns what HandleRemoteMessage was called with
func (mn *MockNotify) HandleRCCalledWith() *remotectrl.Message {
	return mn.rcMsg
}

// HandleRCCalled returns whether HandleRemoteMessage was called
func (mn *MockNotify) HandleRCCalled() bool {
	return mn.rcMsg != nil
}

// Reset resets MockNotify for use again
func (mn *MockNotify) Reset() {
	mn.id = ""
	mn.rcMsg = nil
}
