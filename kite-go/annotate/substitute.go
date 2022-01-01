package annotate

import "strings"

// Substitute all instances of "$xyz$" in s for the corresponding value in vars.
func substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.Replace(s, "$"+k+"$", v, -1)
	}
	return s
}
