package driver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CollectionOrderCompleteness(t *testing.T) {
	for _, p := range prioritizedProviders {
		_, ok := allProviders[p]
		assert.True(t, ok, fmt.Sprintf("The provider %T is missing from all providers (it is present in prioritizedProviders)", p))
	}

	for p := range allProviders {
		found := false
		for _, pr := range prioritizedProviders {
			if p == pr {
				found = true
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("The provider %T is missing from the prioritized providers, it can't be collected", p))
	}
}
