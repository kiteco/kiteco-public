package pythonimports

var (
	interns       = make(map[string]string)
	numInternHits int
)

// get returns a string that contains the same content as s
func intern(s string) string {
	if t, found := interns[s]; found {
		numInternHits++
		return t
	}
	interns[s] = s
	return s
}
