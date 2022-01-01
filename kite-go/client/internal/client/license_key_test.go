package client

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/require"
)

func Test_PublicKey(t *testing.T) {
	key, err := readPublicKey()
	require.NoError(t, err)

	validator := licensing.NewValidatorWithKey(key)
	require.NotNil(t, validator)
}
