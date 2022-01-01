//go:generate protoc --go_out=./ event.proto

package event

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// codeRunCommands contains the command patterns for running/compiling code
var (
	codeRunCommands = map[lang.Language]string{
		lang.Golang: `^go (build|run)`,
		lang.Python: `^python`,
	}
	codeRunRegex = make(map[lang.Language]*regexp.Regexp)
)

// Events is a wrapper around a slice of Event objects.
type Events struct {
	Events []*Event
}

// IsNewText returns whether the given event's text is distinct from the given
// previous event's text.
func IsNewText(ev *Event, prev *Event) bool {
	return ev.GetText() != prev.GetText()
}

// IsNewSel returns whether the given event's selections are distinct from the
// given previous event's selections.
func IsNewSel(ev *Event, prev *Event) bool {
	if len(ev.Selections) != len(prev.Selections) {
		return true
	}
	selMap := make(map[string]bool)
	for _, sel := range prev.GetSelections() {
		s := fmt.Sprintf("%d-%d", sel.GetStart(), sel.GetEnd())
		selMap[s] = true
	}

	for _, sel := range ev.GetSelections() {
		s := fmt.Sprintf("%d-%d", sel.GetStart(), sel.GetEnd())
		_, exists := selMap[s]
		if !exists {
			return true
		}
	}

	return false
}

// IsNewFile returns whether the given event's filename is distinct from the
// given previous event's filename.
func IsNewFile(ev *Event, prev *Event) bool {
	return ev.GetFilename() != prev.GetFilename()
}

// IsTerminal returns whether the given event is from a terminal source.
func IsTerminal(ev *Event) bool {
	return ev.GetSource() == "terminal"
}

// IsTerminalCommand returns whether the given event is for a command from a
// terminal source.
func IsTerminalCommand(ev *Event) bool {
	return IsTerminal(ev) && ev.GetAction() == "command"
}

// IsEditor returns whether the given event is an editor event.
func IsEditor(ev *Event) bool {
	switch ev.GetSource() {
	case "atom", "vim", "nvim", "emacs", "sublime2", "sublime3", "sublime-text", "intellij", "pycharm", "vscode", "bbedit":
		return true
	}
	return false
}

// IsCodeRunCommand returns whether the given event is for a command that
// compiles or runs code
func IsCodeRunCommand(ev *Event) (lang.Language, bool) {
	if !IsTerminalCommand(ev) {
		return lang.Unknown, false
	}
	for l, r := range codeRunRegex {
		s := strings.TrimSpace(ev.GetCommand())
		if r.MatchString(s) {
			return l, true
		}
	}
	return lang.Unknown, false
}

// init sets up the global variables
func init() {
	for l, r := range codeRunCommands {
		codeRunRegex[l] = regexp.MustCompile(r)
	}
}
