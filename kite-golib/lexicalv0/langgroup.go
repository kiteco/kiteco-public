package lexicalv0

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

var (
	// WebGroup ...
	WebGroup = LangGroup{
		Lexer: lang.Text,
		Langs: []lang.Language{lang.JavaScript, lang.JSX, lang.Vue, lang.CSS, lang.HTML, lang.Less, lang.TypeScript, lang.TSX},
	}
	// JavaPlusPlusGroup ...
	JavaPlusPlusGroup = LangGroup{
		Lexer: lang.Text,
		Langs: []lang.Language{lang.Java, lang.Scala, lang.Kotlin},
	}
	// CStyleGroup ...
	CStyleGroup = LangGroup{
		Lexer: lang.Text,
		Langs: []lang.Language{lang.C, lang.Cpp, lang.ObjectiveC, lang.CSharp},
	}
	// MiscLangsGroup ...
	MiscLangsGroup = LangGroup{
		Lexer: lang.Text,
		Langs: []lang.Language{lang.Python, lang.Golang, lang.PHP, lang.Ruby, lang.Bash},
	}
	// AllLangsGroup ...
	AllLangsGroup = LangGroup{
		Lexer: lang.Text,
		Langs: []lang.Language{lang.Python, lang.Golang,
			lang.JavaScript, lang.JSX, lang.Vue, lang.CSS, lang.HTML, lang.Less, lang.TypeScript, lang.TSX,
			lang.Java, lang.Scala, lang.Kotlin,
			lang.C, lang.Cpp, lang.ObjectiveC, lang.CSharp,
			lang.PHP, lang.Ruby, lang.Bash,
		},
	}
)

const (
	baseSep = "__"
	elemSep = "-"
)

// LangGroup ...
type LangGroup struct {
	Lexer lang.Language
	Langs []lang.Language
}

// NewLangGroup ...
func NewLangGroup(lexer lang.Language, langs ...lang.Language) LangGroup {
	if len(langs) == 0 {
		langs = append(langs, lexer)
	}
	return LangGroup{
		Lexer: lexer,
		Langs: langs,
	}
}

// Empty ...
func (g LangGroup) Empty() bool {
	return g.Lexer == lang.Unknown && len(g.Langs) == 0
}

// DeepCopy ...
func (g LangGroup) DeepCopy() LangGroup {
	return LangGroup{
		Lexer: g.Lexer,
		Langs: append([]lang.Language{}, g.Langs...),
	}
}

// Contains ...
func (g LangGroup) Contains(l lang.Language) bool {
	for _, ll := range g.Langs {
		if ll == l {
			return true
		}
	}
	return false
}

// Equals ...
func (g LangGroup) Equals(gg LangGroup) bool {
	if g.Lexer != gg.Lexer {
		return false
	}
	if len(g.Langs) != len(gg.Langs) {
		return false
	}

	for i, l := range g.Langs {
		if l != gg.Langs[i] {
			return false
		}
	}
	return true
}

// IsMultiLingual ...
func (g LangGroup) IsMultiLingual() bool {
	return len(g.Langs) > 1
}

// LangTagFor ...
func (g LangGroup) LangTagFor(path string) int {
	l := lang.FromFilename(path)
	for i, e := range g.Langs {
		if l == e {
			return i
		}
	}

	return -1
}

// MustLangTagFor ...
func (g LangGroup) MustLangTagFor(path string) int {
	t := g.LangTagFor(path)
	if t == -1 {
		panic(fmt.Sprintf("unable to get lang tag for path %s with group %s", path, g.Name()))
	}
	return t
}

// Name ...
func (g LangGroup) Name() string {
	// TODO: backwards compat hack, see LangGroupFromName below
	if len(g.Langs) == 1 && g.Langs[0] == g.Lexer {
		return g.Lexer.Name()
	}

	var langs []string
	for _, l := range g.Langs {
		langs = append(langs, l.Name())
	}
	return fmt.Sprintf("%s%s%s", g.Lexer.Name(), baseSep, strings.Join(langs, elemSep))
}

// LangGroupFromName ...
func LangGroupFromName(s string) (LangGroup, error) {
	parts := strings.Split(s, baseSep)
	switch len(parts) {
	case 1, 2:
		base := lang.FromName(parts[0])
		if base == lang.Unknown {
			return LangGroup{}, errors.New("unable to parse base base lang from %s", base)
		}

		if len(parts) == 1 {
			// backwards compat, see Name above
			return NewLangGroup(base, base), nil
		}

		var langs []lang.Language
		for _, part := range strings.Split(parts[1], elemSep) {
			part = strings.TrimSpace(part)
			l := lang.FromName(part)
			if l == lang.Unknown {
				return LangGroup{}, errors.New("unable to parse elem '%s' from %s", part, s)
			}

			for _, ll := range langs {
				if l == ll {
					return LangGroup{}, errors.New("got language %s more than onece in %s", l.Name(), s)
				}
			}
			langs = append(langs, l)
		}

		return NewLangGroup(base, langs...), nil
	default:
		return LangGroup{}, errors.New("invalid name '%s', must contain 0 or 1 instances of %s", s, baseSep)
	}
}

// MustLangGroupFromName ...
func MustLangGroupFromName(s string) LangGroup {
	g, err := LangGroupFromName(s)
	if err != nil {
		panic(err)
	}
	return g
}
