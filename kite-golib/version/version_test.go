package version

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfo_LargerThanOrEqualTo(t *testing.T) {
	assertLargerOrEqual(t, "0", "0")
	assertLargerOrEqual(t, "0.1", "0.0.1")

	assertLargerOrEqual(t, "1", "0")
	assertLargerOrEqual(t, "1", "0.9")
	assertLargerOrEqual(t, "1", "0.99")
	assertLargerOrEqual(t, "1", "1")
	assertLargerOrEqual(t, "1", "1.0")
	assertLargerOrEqual(t, "1.9.0", "1.9.0")
	assertLargerOrEqual(t, "1.9.0", "1.9.0")

	assertLargerOrEqual(t, "2.1.3", "1")
	assertLargerOrEqual(t, "2.1.3", "2")
	assertLargerOrEqual(t, "2.1.3", "2.1")
	assertLargerOrEqual(t, "2.1.3", "2.1.2")
	assertLargerOrEqual(t, "2.1.3", "2.1.2.2")

	assertLessThan(t, "0", "1")
	assertLessThan(t, "0", "100")
	assertLessThan(t, "1.0", "2")
	assertLessThan(t, "1.0", "2.0")
	assertLessThan(t, "1.0", "1.0.1")
	assertLessThan(t, "1.0", "1.1")
	assertLessThan(t, "1.0", "1.1.0")
}

func TestInfo_LargerThan(t *testing.T) {
	assertLarger(t, "1", "0")
	assertLarger(t, "0.1", "0.0.1")
	assertNotLarger(t, "1", "2")
	assertNotLarger(t, "0.1", "0.2")
	assertNotLarger(t, "0.0.1", "0.1")

	assertLarger(t, "1", "0")
	assertLarger(t, "1", "0.9")
	assertLarger(t, "1", "0.99")
	assertNotLarger(t, "1", "1")
	assertNotLarger(t, "1", "1.0")
	assertNotLarger(t, "1.9.0", "1.9.0")
	assertNotLarger(t, "1.9.0", "1.9.0")

	assertLarger(t, "2.1.3", "1")
	assertLarger(t, "2.1.3", "2")
	assertLarger(t, "2.1.3", "2.1")
	assertLarger(t, "2.1.3", "2.1.2")
	assertLarger(t, "2.1.3", "2.1.2.2")
}

func TestInfo_String(t *testing.T) {
	v, err := Parse("2.1.0")
	assert.NoError(t, err)

	if v.String() != "2.1.0" {
		t.Errorf("Version must be printed as string: %s", v)
	}
}

func TestInfo_Suffix(t *testing.T) {
	v, err := Parse("2-a.b.c")
	assert.NoError(t, err)

	v, err = Parse("0.2.1-25-g2e3b78d1")
	assert.NoError(t, err)

	v, err = Parse("2.3.4-a.b.c")
	assert.NoError(t, err)
	if v.Patch() != 4 {
		t.Errorf("Patch must equal 4: %d", v.Minor())
	}
	if v.Suffix != "-a.b.c" {
		t.Error("Minor must equal '-a.b.c'")
	}
}

func TestInfo_Error(t *testing.T) {
	_, err := Parse("1.x")
	assert.Error(t, err)

	_, err = Parse("1.2.x")
	assert.Error(t, err)

	_, err = Parse(".2-x")
	assert.Error(t, err)
}

func assertLargerOrEqual(t *testing.T, a string, b string) {
	v1, err := Parse(a)
	assert.NoError(t, err)

	v2, err := Parse(b)
	assert.NoError(t, err)

	if !v1.LargerThanOrEqualTo(v2) {
		t.Errorf("%s must be larger than or equal to %s", a, b)
	}
}

func assertLarger(t *testing.T, a string, b string) {
	v1, err := Parse(a)
	assert.NoError(t, err)

	v2, err := Parse(b)
	assert.NoError(t, err)

	if !v1.LargerThan(v2) {
		t.Errorf("%s must be larger than %s", a, b)
	}
}

func assertNotLarger(t *testing.T, a string, b string) {
	v1, err := Parse(a)
	assert.NoError(t, err)

	v2, err := Parse(b)
	assert.NoError(t, err)

	if v1.LargerThan(v2) {
		t.Errorf("%s must not be larger than %s", a, b)
	}
}

func assertLessThan(t *testing.T, a string, b string) {
	v1, err := Parse(a)
	assert.NoError(t, err)

	v2, err := Parse(b)
	assert.NoError(t, err)

	if v1.LargerThanOrEqualTo(v2) {
		t.Errorf("%s must be less than %s", a, b)
	}
}

func requireInfo(t *testing.T, v string) Info {
	parsed, err := Parse(v)
	require.NoError(t, err)
	return parsed
}

func requireInfos(t *testing.T, vs ...string) Infos {
	var infos []Info
	for _, v := range vs {
		infos = append(infos, requireInfo(t, v))
	}
	return infos
}

func TestInfos(t *testing.T) {
	infos := requireInfos(t, "1.2", "3.2.3-2", "1.2.3", "3.2.3-1", "1.0", "3.2.3", "2.4", "1.0.1")

	sort.Sort(infos)

	expected := []string{
		"1.0",
		"1.0.1",
		"1.2",
		"1.2.3",
		"2.4",
		"3.2.3",
		"3.2.3-1",
		"3.2.3-2",
	}

	var actual []string
	for _, info := range infos {
		actual = append(actual, info.String())
	}

	assert.Equal(t, expected, actual)
}
