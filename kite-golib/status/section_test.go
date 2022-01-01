package status

import (
	"testing"
	"time"
)

func TestSection_MarshalJSON_NoInfiniteLoop(t *testing.T) {
	// This test just checks that Section.MarshalJSON does not go into an infinite loop.
	// See the comment within that function for why this is worth testing.
	ch := make(chan struct{})
	go func() {
		var s Section
		s.MarshalJSON()
		close(ch)
	}()
	select {
	case <-ch:
		// MarshalJSON terminated correctly -> test passed
	case <-time.After(time.Second):
		t.Error("Section.MarshalJSON did not terminate")
	}
}
