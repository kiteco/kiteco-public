package data

import "path/filepath"

// ProviderName defines a name of a particular Provider as an enum
type ProviderName int

// Defining provider name enum
const (
	// Python Providers
	// PythonEmptyAttrsProvider is retired
	PythonEmptyAttrsProvider ProviderName = iota + 1
	PythonEmptyCallsProvider
	PythonCallPatternsProvider
	PythonImportsProvider
	PythonAttributesProvider
	PythonNamesProvider
	PythonKeywordsProvider
	PythonCallModelProvider
	PythonAttributeModelProvider
	PythonExprModelProvider
	PythonKWArgsProvider
	PythonDictKeysProvider
	PythonGGNNModelProvider
	// Wraps LexicalPythonProvider
	PythonLexicalProvider

	// Lexical Providers
	LexicalGolangProvider
	LexicalJavascriptProvider
	LexicalPythonProvider

	LexicalTextGolangProvider

	// Lexical All Language (by extension names)
	LexicalTextCProvider
	LexicalTextCCProvider
	LexicalTextCPPProvider
	LexicalTextCSProvider
	LexicalTextCSSProvider
	LexicalTextHProvider
	LexicalTextHPPProvider
	LexicalTextHTMLProvider
	LexicalTextJAVAProvider
	LexicalTextJSProvider
	LexicalTextJSXProvider
	LexicalTextKTProvider
	LexicalTextLESSProvider
	LexicalTextMProvider
	LexicalTextPHPProvider
	LexicalTextRBProvider
	LexicalTextSCALAProvider
	LexicalTextSHProvider
	LexicalTextTSProvider
	LexicalTextTSXProvider
	LexicalTextVUEProvider

	// Generic text provider to use for /lexicalproviders/TextProvider.Name()
	LexicalTextProvider
)

func (p ProviderName) String() string {
	if val, ok := providerNameStrings[p]; ok {
		return val
	}
	return "UnknownProvider"
}

var providerNameStrings = map[ProviderName]string{
	// Python Providers
	PythonEmptyAttrsProvider:     "PythonEmptyAttrsProvider",
	PythonEmptyCallsProvider:     "PythonEmptyCallsProvider",
	PythonCallPatternsProvider:   "PythonCallPatternsProvider",
	PythonImportsProvider:        "PythonImportsProvider",
	PythonAttributesProvider:     "PythonAttributesProvider",
	PythonNamesProvider:          "PythonNamesProvider",
	PythonKeywordsProvider:       "PythonKeywordsProvider",
	PythonCallModelProvider:      "PythonCallModelProvider",
	PythonAttributeModelProvider: "PythonAttributeModelProvider",
	PythonExprModelProvider:      "PythonExprModelProvider",
	PythonKWArgsProvider:         "PythonKWArgsProvider",
	PythonDictKeysProvider:       "PythonDictKeysProvider",
	PythonGGNNModelProvider:      "PythonGGNNModelProvider",
	PythonLexicalProvider:        "PythonLexicalProvider",

	// Lexical Providers
	LexicalGolangProvider:     "LexicalGolangProvider",
	LexicalJavascriptProvider: "LexicalJavascriptProvider",
	LexicalPythonProvider:     "LexicalPythonProvider",
	LexicalTextGolangProvider: "LexicalTextGolangProvider",

	// Lexical All Language Providers
	LexicalTextCProvider:     "LexicalTextCProvider",
	LexicalTextCCProvider:    "LexicalTextCCProvider",
	LexicalTextCPPProvider:   "LexicalTextCPPProvider",
	LexicalTextCSProvider:    "LexicalTextCSProvider",
	LexicalTextCSSProvider:   "LexicalTextCSSProvider",
	LexicalTextHProvider:     "LexicalTextHProvider",
	LexicalTextHPPProvider:   "LexicalTextHPPProvider",
	LexicalTextHTMLProvider:  "LexicalTextHTMLProvider",
	LexicalTextJAVAProvider:  "LexicalTextJAVAProvider",
	LexicalTextJSProvider:    "LexicalTextJSProvider",
	LexicalTextJSXProvider:   "LexicalTextJSXProvider",
	LexicalTextKTProvider:    "LexicalTextKTProvider",
	LexicalTextLESSProvider:  "LexicalTextLESSProvider",
	LexicalTextMProvider:     "LexicalTextMProvider",
	LexicalTextPHPProvider:   "LexicalTextPHPProvider",
	LexicalTextRBProvider:    "LexicalTextRBProvider",
	LexicalTextSCALAProvider: "LexicalTextSCALAProvider",
	LexicalTextSHProvider:    "LexicalTextSHProvider",
	LexicalTextTSProvider:    "LexicalTextTSProvider",
	LexicalTextTSXProvider:   "LexicalTextTSXProvider",
	LexicalTextVUEProvider:   "LexicalTextVUEProvider",

	LexicalTextProvider: "LexicalTextProvider",
}

// TextProviderNameFromPath ...
func TextProviderNameFromPath(path string) ProviderName {
	var ext string
	if e := filepath.Ext(path); e != "" {
		ext = e[1:]
	}
	if val, ok := extToTextProvider[ext]; ok {
		return val
	}
	return -1
}

var extToTextProvider = map[string]ProviderName{
	"c":     LexicalTextCProvider,
	"cc":    LexicalTextCCProvider,
	"cpp":   LexicalTextCPPProvider,
	"cs":    LexicalTextCSProvider,
	"css":   LexicalTextCSSProvider,
	"go":    LexicalGolangProvider,
	"h":     LexicalTextHProvider,
	"hpp":   LexicalTextHPPProvider,
	"html":  LexicalTextHTMLProvider,
	"java":  LexicalTextJAVAProvider,
	"js":    LexicalTextJSProvider,
	"jsx":   LexicalTextJSXProvider,
	"kt":    LexicalTextKTProvider,
	"less":  LexicalTextLESSProvider,
	"m":     LexicalTextMProvider,
	"php":   LexicalTextPHPProvider,
	"rb":    LexicalTextRBProvider,
	"scala": LexicalTextSCALAProvider,
	"sh":    LexicalTextSHProvider,
	"ts":    LexicalTextTSProvider,
	"tsx":   LexicalTextTSXProvider,
	"vue":   LexicalTextVUEProvider,
}

// ProviderNotApplicableError is an error returned by a Provider indicating the Provider is not applicable to the given completions situation
type ProviderNotApplicableError struct{}

// Error implements error
func (e ProviderNotApplicableError) Error() string {
	return "Provider not applicable"
}
