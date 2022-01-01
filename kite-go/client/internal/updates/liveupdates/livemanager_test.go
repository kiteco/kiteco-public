package liveupdates

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

func Test_ComponentInterfaces(t *testing.T) {
	m := &LiveManager{}
	_ = component.Handlers(m)
	_ = component.ProcessedEventer(m)
	_ = component.Terminater(m)
}
