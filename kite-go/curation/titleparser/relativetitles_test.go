package titleparser

import "testing"

var (
	test1 = "Construct a 1D array [of `int16`] [of ones]"
	test2 = "[of ones]"
	test3 = "Construct a 1D array"
)

func TestBasicParse(t *testing.T) {
	parser := newBasicParser()

	expected1 := []string{"Construct", "a 1D array", "of `int16`", "of ones"}
	expected2 := []string{}
	expected3 := []string{"Construct", "a 1D array"}

	var res1, res2, res3 []string
	for _, p := range parser.parse(test1) {
		res1 = append(res1, p.text)
	}
	for _, p := range parser.parse(test2) {
		res2 = append(res2, p.text)
	}
	for _, p := range parser.parse(test3) {
		res3 = append(res3, p.text)
	}

	if !compareStrings(res1, expected1) {
		t.Errorf("test1 is not parsed correctly.\n")
	}

	if !compareStrings(res2, expected2) {
		t.Errorf("test2 is not parsed correctly.\n")
	}

	if !compareStrings(res3, expected3) {
		t.Errorf("test3 is not parsed correctly.\n")
	}
}

func compareStrings(res, exp []string) bool {
	if len(res) != len(exp) {
		return false
	}

	for i, s := range res {
		if s != exp[i] {
			return false
		}
	}
	return true
}

func TestRelativeTitles(t *testing.T) {
	exp := "Construct a 1D array <b>of int16</b> <b>of ones</b>"
	res := RelativeTitle(test1, test3)
	if res != exp {
		t.Errorf("Expected relative title to be '%s', got '%s'\n", exp, res)
	}

	exp = "Construct a 1D array"
	res = RelativeTitle(test3, test1)
	if res != exp {
		t.Errorf("Expected relative title to be '%s', got '%s'\n", exp, res)
	}
}
