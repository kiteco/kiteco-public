package pythondocs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AddNoFollow(t *testing.T) {
	in := `
<div>
    <p>paragraph text</p>
    <a href="#nofollow">skip</a>
    <a class = "internal_link" href="#nofollow">no follow</a>
</div>
<a class="internal_link" href="#follow">
`
	expected := `
<div>
    <p>paragraph text</p>
    <a href="#nofollow">skip</a>
    <a class="internal_link" href="#nofollow" rel="nofollow">no follow</a>
</div>
<a class="internal_link" href="#follow">
`
	actual := AddNoFollow(in, func(id string) bool {
		return id == "nofollow"
	})

	require.Equal(t, expected, actual)
}
