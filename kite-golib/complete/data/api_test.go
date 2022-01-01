package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_EventOffsetEncoding(t *testing.T) {
	req := APIRequest{
		APIOptions: APIOptions{
			Editor: "intellij",
		},
		SelectedBuffer: SelectedBuffer{
			Buffer: "print(\"史史史史史史史史史史\")\n",
			Selection: Selection{
				Begin: 40,
				End:   40,
			},
		},
	}

	err := req.Validate()
	require.NoError(t, err)
}
