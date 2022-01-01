package pythondocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	index := map[string][]string{
		"os":   []string{"os.path.join", "os.test.path.join"},
		"path": []string{"re.path", "os.path.join", "os.test.path.join"},
		"join": []string{"os.path.join", "re.join", "os.test.path.join"},
		"test": []string{"os.test.path.join"},
		"re":   []string{"re.path", "re.join"},
	}

	nameToID := map[string]int64{
		"os.test.path.join": 1,
		"os.path.join":      2,
		"re.path":           3,
		"re.join":           4,
	}

	table := &IdentifierLookupTable{
		index:    index,
		nameToID: nameToID,
	}

	require.Equal(t, 2, len(table.Find("os path")))
	assert.Equal(t, "os.path.join", table.Find("os path")[0])

	require.Equal(t, 2, len(table.Find("os join")))
	assert.Equal(t, "os.path.join", table.Find("os join")[0])
}
