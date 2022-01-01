package pythonscanner

// PythonKeywordPrefixes is a set containing all the
// possible prefixes for any python keyword.
var PythonKeywordPrefixes map[string]struct{}

func init() {
	PythonKeywordPrefixes = make(map[string]struct{})
	for word := range Keywords {
		for i := 1; i <= len(word); i++ {
			prefix := word[0:i]
			PythonKeywordPrefixes[prefix] = struct{}{}
		}
	}
}
