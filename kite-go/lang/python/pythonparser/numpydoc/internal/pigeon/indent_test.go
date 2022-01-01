package pigeon

import (
	"reflect"
	"strings"
	"testing"
)

func TestIndent(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"", []int{0}},
		{" ", []int{0, 1}},
		{"  ", []int{0, 2}},
		{"\t", []int{0, 8}},
		{" \t", []int{0, 8}},
		{"  \t", []int{0, 8}},
		{"   \t", []int{0, 8}},
		{"    \t", []int{0, 8}},
		{"     \t", []int{0, 8}},
		{"      \t", []int{0, 8}},
		{"       \t", []int{0, 8}},
		{"        \t", []int{0, 16}},
		{" \t ", []int{0, 9}},
		{"\t ", []int{0, 9}},
		{"\t  ", []int{0, 10}},
		{"\t   ", []int{0, 11}},
		{"\t    ", []int{0, 12}},
		{"\t     ", []int{0, 13}},
		{"\t      ", []int{0, 14}},
		{"\t       ", []int{0, 15}},
		{"\t        ", []int{0, 16}},
		{"\t       \t", []int{0, 16}},
		{"\t        \t", []int{0, 24}},
		{"\t\t", []int{0, 16}},
		{"\t\t ", []int{0, 17}},
		{"\t\t \t", []int{0, 24}},

		{" \n  \n   \n    ", []int{0, 1, 2, 3, 4}},
		{" \n  \n   \n    \n   ", []int{0, 1, 2, 3}},
		{" \n  \n   \n    \n  ", []int{0, 1, 2}},
		{" \n  \n   \n    \n ", []int{0, 1}},
		{" \n  \n   \n    \n", []int{0}},

		{"\t\n        ", []int{0, 8}},
		{" \n \n \n ", []int{0, 1}},
		{" \n \n \n\t", []int{0, 1, 8}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			// create and initialize *current and its state
			cur := &current{state: storeDict{}}
			if err := initState(cur); err != nil {
				t.Fatal(err)
			}

			// process all indentations
			lines := strings.Split(c.in, "\n")
			for _, line := range lines {
				indent := stringToIfaceSlice(line)
				if err := indentState(cur, indent); err != nil {
					t.Fatal(err)
				}
			}

			// assert the state of the stack
			stack := cur.state[indentStackKey].(indentStack)
			stackIndex := cur.state[indentStackIndexKey].(int)
			got := stack[:stackIndex]

			if !reflect.DeepEqual(c.want, got) {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func stringToIfaceSlice(s string) []interface{} {
	vs := make([]interface{}, len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		vs[i] = []byte{b}
	}
	return vs
}
