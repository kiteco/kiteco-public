package email

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseAddress(t *testing.T) {
	type parseTest struct {
		input  string
		user   string
		canon  string
		domain string
		err    bool
	}

	testCases := []parseTest{
		parseTest{"foo@example.com", "foo", "foo", "example.com", false},
		parseTest{"foo+stuff@example.com", "foo+stuff", "foo", "example.com", false},
		parseTest{"foo.bar@example.com", "foo.bar", "foobar", "example.com", false},
		parseTest{"foo.bar+stuff@example.com", "foo.bar+stuff", "foobar", "example.com", false},
		parseTest{"fooexample.com", "", "", "", true},
	}

	for _, tc := range testCases {
		addr, err := ParseAddress(tc.input)
		if tc.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tc.user, addr.User)
		require.Equal(t, tc.canon, addr.canonicalUser())
		require.Equal(t, tc.domain, addr.Domain)
	}
}

func Test_Duplicate(t *testing.T) {
	type duplicateTest struct {
		input1 string
		input2 string
		equal  bool
	}

	testCases := []duplicateTest{
		// Basics
		duplicateTest{"foo@example.com", "foo@example.com", true},
		duplicateTest{"foo@example.com", "foo@example2.com", false},
		duplicateTest{"foo@example.com", "bar@example.com", false},

		// + things
		duplicateTest{"foo+hi@example.com", "foo+hi@example.com", true},
		duplicateTest{"foo+hi@example.com", "foo+bye@example.com", true},
		duplicateTest{"foo+hi@example.com", "foo+bye@example.com", true},

		// . things
		duplicateTest{"foo.hi@example.com", "foohi@example.com", true},
		duplicateTest{"foo.hi@example.com", "foo.hi@example.com", true},
		duplicateTest{"foo.hi@example.com", "foo@example.com", false},

		// Together now!...
		duplicateTest{"foo.hi+one@example.com", "foohi+two@example.com", true},
		duplicateTest{"foo.hi+two@example.com", "foo.hi@example.com", true},
		duplicateTest{"foo.hi@example.com", "foo.bye@example.com", false},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.equal, Duplicate(tc.input1, tc.input2))
	}
}
