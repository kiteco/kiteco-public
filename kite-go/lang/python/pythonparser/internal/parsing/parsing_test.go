package parsing

import "testing"

func TestTrimMaxLines(t *testing.T) {
	cases := []struct {
		in, out string
		max     uint64
	}{
		{"", "", 0},
		{"", "", 1},

		{"a", "a", 0},
		{"a", "a", 1},

		{"a\n", "a\n", 0},
		{"a\n", "a", 1},

		{"a\n\n", "a\n\n", 0},
		{"a\n\n", "a", 1},
		{"a\n\n", "a\n", 2},

		{"a\r\n", "a\r\n", 0},
		{"a\r\n", "a", 1},
		{"a\r\n", "a\r\n", 2},

		{"a\r\r", "a\r\r", 0},
		{"a\r\r", "a", 1},
		{"a\r\r", "a\r", 2},

		{"a\r\n\n", "a\r\n\n", 0},
		{"a\r\n\n", "a", 1},
		{"a\r\n\n", "a\r\n", 2},
		{"a\r\n\n", "a\r\n\n", 3},

		{"a\r\n\r", "a\r\n\r", 0},
		{"a\r\n\r", "a", 1},
		{"a\r\n\r", "a\r\n", 2},
		{"a\r\n\r", "a\r\n\r", 3},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			out, trimmed := TrimMaxLines([]byte(c.in), c.max)
			if got := string(out); c.out != got {
				t.Fatalf("want %q, got %q", c.out, got)
			}
			wantTrimmed := c.in != c.out
			if trimmed != wantTrimmed {
				t.Fatalf("want trimmed %t, got %t", wantTrimmed, trimmed)
			}
		})
	}
}
