package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Validation(t *testing.T) {
	for key, spec := range specs {
		if spec.defaultValue != nil && spec.validate != nil {
			err := spec.validate(*spec.defaultValue)
			require.NoErrorf(t, err, "default value for %s not valid: %v", key, err)
		}
	}
}
