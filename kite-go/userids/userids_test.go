package userids

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserIds(t *testing.T) {
	ids := NewUserIDs("install", "machine")
	ids.SetUser(42, "email", true)
	assert.EqualValues(t, "42", ids.MetricsID(), "the id used for metrics must return the user id when available")

	ids = NewUserIDs("install", "machine")
	ids.SetUser(0, "email", true)
	assert.EqualValues(t, "install", ids.MetricsID(), "the id used for metrics must return the install id when no user id is available")

	ids = NewUserIDs("install", "machine")
	ids.SetUser(42, "email", false)
	assert.EqualValues(t, "install", ids.ForgetfulMetricsID(), "the forgetful metrics ID must return the install id when not logged in")

	ids = NewUserIDs("", "")
	assert.Empty(t, ids.MetricsID(), "the ids must be empty when no id value is present")
}

func TestUpdate(t *testing.T) {
	ids := NewUserIDs("install", "machine")
	ids.SetUser(42, "email", true)
	assert.EqualValues(t, 42, ids.UserID())
	assert.EqualValues(t, "email", ids.Email())
	assert.EqualValues(t, "install", ids.InstallID())
	assert.EqualValues(t, "machine", ids.MachineID())

	ids.SetUser(1024, "new email", false)
	assert.EqualValues(t, 1024, ids.UserID())
	assert.EqualValues(t, "new email", ids.Email())
}
