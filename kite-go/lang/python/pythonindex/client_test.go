package pythonindex

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// We only use this for testing purpose.
func newIdentCount(ident string, count int) *IdentCount {
	return &IdentCount{
		Ident:       ident,
		Count:       count,
		ForcedCount: count,
	}
}

func makeClient(nodes []*IdentCount) *Client {
	invertedIndex := make(map[string][]*IdentCount)
	for _, n := range nodes {
		for _, part := range strings.Split(n.Ident, ".") {
			invertedIndex[part] = append(invertedIndex[part], n)
		}
	}

	opts := ClientOptions{
		MinCoverage:   1,
		MinOccurrence: 0,
	}

	return &Client{
		packageStats: &index{
			invertedIndex: invertedIndex,
		},
		opts: &opts,
	}
}

func TestSearchWithCount(t *testing.T) {
	client := makeClient(
		[]*IdentCount{
			newIdentCount("os.path.join", 100),
			newIdentCount("os.test.join", 50),
			newIdentCount("test.join", 10),
			newIdentCount("market.join", 5),
			newIdentCount("json.loads", 1),
			newIdentCount("mock.path.join", 200),
		})

	r := client.SearchWithCount("join")
	require.Len(t, r.IdentCounts, 5)

	assert.Equal(t, IdentCount{"mock.path.join", 200, 200, ""}, *r.IdentCounts[0])
	assert.Equal(t, IdentCount{"os.path.join", 100, 100, ""}, *r.IdentCounts[1])
	assert.Equal(t, IdentCount{"os.test.join", 50, 50, ""}, *r.IdentCounts[2])
}

func TestSearch(t *testing.T) {
	client := makeClient(
		[]*IdentCount{
			newIdentCount("os.path.join", 100),
			newIdentCount("os.test.join", 50),
			newIdentCount("test.join", 10),
			newIdentCount("market.join", 5),
			newIdentCount("json.loads", 1),
			newIdentCount("mock.path.join", 200),
		})

	names := client.Search("os join")
	require.Len(t, names, 2)

	assert.Equal(t, "os.path.join", names[0])
	assert.Equal(t, "os.test.join", names[1])

	names = client.Search("OS join")
	require.Len(t, names, 2)

	names = client.Search("join")
	require.Len(t, names, 5)
}

func TestCompletion(t *testing.T) {
	client := makeClient(
		[]*IdentCount{
			newIdentCount("mock.path.join", 200),
			newIdentCount("json.load", 250),
			newIdentCount("os.path.join", 100),
			newIdentCount("os.test.join", 50),
			newIdentCount("test.join", 10),
			newIdentCount("market.join", 5),
			newIdentCount("json.loads", 1),
		})

	// We build suffix array only from identifier parts not from example titles or doc strings.
	var tokens []string
	for t := range client.packageStats.invertedIndex {
		tokens = append(tokens, t)
	}

	client.suffixArray = newSuffixArray(tokens)

	completions := client.QueryCompletion("json lo")
	require.Len(t, completions, 2)

	assert.Equal(t, "json.load", completions[0].Display)
	assert.Equal(t, "json.loads", completions[1].Display)

	completions = client.QueryCompletion("os j")
	require.Len(t, completions, 2)

	assert.Equal(t, "os.path.join", completions[0].Display)
	assert.Equal(t, "os.test.join", completions[1].Display)

	completions = client.QueryCompletion("j")
	require.Len(t, completions, 6)

	assert.Equal(t, "json.load", completions[0].Display)
	assert.Equal(t, "os.path.join", completions[2].Display)
}
