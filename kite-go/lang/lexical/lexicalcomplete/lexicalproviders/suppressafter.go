package lexicalproviders

func createSuppressMap(l []string) map[string]bool {
	m := make(map[string]bool)
	m["\n"] = true
	for _, s := range l {
		m[s] = true
	}
	return m
}

var golangSuppressAfter = createSuppressMap([]string{")", "{", "}", ";", ","})
var javascriptSuppressAfter = createSuppressMap([]string{")", "{", "}", ";", ",", ">"})
var pythonSuppressAfter = createSuppressMap([]string{")", "]", "}", "\"", "'", ":", "\\"})
var pythonMaybeSuppressAfter = createSuppressMap([]string{"{", "[", "("})

// For Text Provider, by extension
// Using threshold 0.15 for probability of appearing at end of line
var textSuppressAfter = map[string]map[string]bool{
	"c":     createSuppressMap([]string{";", "{", "}", ":", ")", ","}),
	"cc":    createSuppressMap([]string{";", "{", "}", ","}),
	"cpp":   createSuppressMap([]string{";", "{", "}", ")"}),
	"cs":    createSuppressMap([]string{";", "{", "}", ">", "]", ")"}),
	"css":   createSuppressMap([]string{"{", ";", "}", ","}),
	"h":     createSuppressMap([]string{";", "{", "}", "\\", ".", ">", "\""}),
	"hpp":   createSuppressMap([]string{";", "{", "}", "\\", ">", ".", ")"}),
	"html":  createSuppressMap([]string{"{", "}", ">", ";"}),
	"java":  createSuppressMap([]string{";", "{", "}", ">", ":"}),
	"kt":    createSuppressMap([]string{";", "{", "}", ")", ",", "+"}),
	"less":  createSuppressMap([]string{";", "{", "}", ","}),
	"m":     createSuppressMap([]string{";", "{", "}", ">"}),
	"php":   createSuppressMap([]string{";", "{", "}", ",", ">"}),
	"rb":    createSuppressMap([]string{")", "}", "?", "|", ";", "]", "!", ","}),
	"scala": createSuppressMap([]string{";", "{", "}", ">", ")", "?"}),
	"sh":    createSuppressMap([]string{")", "\\", "`", ";", "}", "\"", "?"}),
	"ts":    createSuppressMap([]string{">", "{", ";", "}", ","}),
	"tsx":   createSuppressMap([]string{";", ">", "{", "}", ",", "`"}),
}
