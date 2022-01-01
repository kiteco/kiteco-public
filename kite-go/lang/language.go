package lang

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Language is an identifier for a supported programming language.
type Language int

// The languages identified by Language.
const (
	Unknown Language = iota
	Golang
	JavaScript
	Cpp
	Java
	Python
	PHP
	ObjectiveC
	Scala
	C
	CSharp
	Perl
	Ruby
	Bash
	HTML
	Less
	CSS
	JSX
	Vue
	TypeScript
	Kotlin
	TSX
	Text
)

// Tags holds a language name and the filename extension for its source files.
type Tags struct {
	Ext     string
	Name    string
	Shebang []string // substrings to look for in the shebang ("#!...")
	Exts    []string
}

// File extensions and names of supported programming languages.
var (
	LanguageTags = map[Language]Tags{
		Golang:     Tags{"go", "go", nil, []string{"go"}},
		JavaScript: Tags{"js", "javascript", nil, []string{"js"}},
		Cpp:        Tags{"cpp", "cpp", nil, []string{"cpp", "cc", "h", "hpp"}},
		Java:       Tags{"java", "java", nil, []string{"java"}},
		Python:     Tags{"py", "python", []string{"python"}, []string{"py", "pyw", "pyt"}},
		PHP:        Tags{"php", "php", []string{"php"}, []string{"php"}},
		ObjectiveC: Tags{"m", "objectivec", nil, []string{"m", "h"}},
		Scala:      Tags{"scala", "scala", nil, []string{"scala"}},
		C:          Tags{"c", "c", nil, []string{"c", "h"}},
		CSharp:     Tags{"cs", "csharp", nil, []string{"cs", "h"}},
		Perl:       Tags{"pl", "perl", []string{"perl"}, []string{"pl"}},
		Ruby:       Tags{"rb", "ruby", []string{"ruby"}, []string{"rb"}},
		Bash:       Tags{"sh", "bash", []string{"bash"}, []string{"sh"}},
		HTML:       Tags{"html", "html", nil, []string{"html"}},
		Less:       Tags{"less", "less", nil, []string{"less"}},
		CSS:        Tags{"css", "css", nil, []string{"css"}},
		JSX:        Tags{"jsx", "jsx", nil, []string{"jsx"}},
		Vue:        Tags{"vue", "vue", nil, []string{"vue"}},
		TypeScript: Tags{"ts", "typescript", nil, []string{"ts"}},
		Kotlin:     Tags{"kt", "kotlin", nil, []string{"kt"}},
		TSX:        Tags{"tsx", "tsx", nil, []string{"tsx"}},
		Text:       Tags{"", "text", nil, []string{""}},
	}
)

var langsSorted []Language

func init() {
	for l := range LanguageTags {
		langsSorted = append(langsSorted, l)
	}
	sort.Slice(langsSorted, func(i, j int) bool {
		return langsSorted[i] < langsSorted[j]
	})
}

// Name returns the name of this language ("javascript", "csharp", etc)
func (lang Language) Name() string {
	switch lang {
	case Golang:
		return "go"
	case JavaScript:
		return "javascript"
	case Cpp:
		return "cpp"
	case Java:
		return "java"
	case Python:
		return "python"
	case PHP:
		return "php"
	case ObjectiveC:
		return "objectivec"
	case Scala:
		return "scala"
	case C:
		return "c"
	case CSharp:
		return "csharp"
	case Perl:
		return "perl"
	case Ruby:
		return "ruby"
	case Bash:
		return "bash"
	case HTML:
		return "html"
	case Less:
		return "less"
	case CSS:
		return "css"
	case JSX:
		return "jsx"
	case Vue:
		return "vue"
	case TypeScript:
		return "typescript"
	case Kotlin:
		return "kotlin"
	case TSX:
		return "tsx"
	case Text:
		return "text"
	default:
		return "unknown"
	}
}

// Extension returns the standard file extension for this language ("js", "cs", etc)
func (lang Language) Extension() string {
	return LanguageTags[lang].Ext
}

// Extensions returns the list of possible file extensions for this language,
// this list always contains atleast `lang.Extension()`
func (lang Language) Extensions() []string {
	return LanguageTags[lang].Exts
}

// IsSupportedLanguage tests whether a given string is the name of a supported
// language (ie, appears in lang.LanguageTags)
func IsSupportedLanguage(name string) bool {
	for _, tag := range LanguageTags {
		if name == tag.Name {
			return true
		}
	}
	return false
}

// MustFromName gets the language with a given name or panics
func MustFromName(name string) Language {
	l := FromName(name)
	if l == Unknown {
		panic(fmt.Sprintf("unsupported language name '%s'", name))
	}
	return l
}

// FromName gets the language with a given name, or Unknown
func FromName(name string) Language {
	for l, tag := range LanguageTags {
		if tag.Name == name {
			return l
		}
	}
	return Unknown
}

// FromExtension gets the language with a given extension, or Unknown
func FromExtension(ext string) Language {
	for l, tag := range LanguageTags {
		if tag.Ext == ext {
			return l
		}
	}
	for _, l := range langsSorted {
		for _, e := range LanguageTags[l].Exts {
			if e == ext {
				return l
			}
		}
	}
	return Unknown
}

// FromFilename returns lang.Language given a filename.
func FromFilename(filename string) Language {
	ext := filepath.Ext(filename)
	if ext == "" {
		return Unknown
	}
	ext = ext[1:]
	return FromExtension(ext)
}

// FromFilenameAndContent returns lang.Language given a filename and its first line.
func FromFilenameAndContent(filename, content string) Language {
	if l := FromFilename(filename); l != Unknown && l != Text {
		return l
	}
	// try to match the shebang
	if strings.HasPrefix(content, "#!") {
		firstLine := content
		if pos := strings.Index(content, "\n"); pos != -1 {
			firstLine = content[:pos]
		}
		for l, tag := range LanguageTags {
			for _, substr := range tag.Shebang {
				if strings.Contains(firstLine, substr) {
					return l
				}
			}
		}
	}
	return Unknown
}
