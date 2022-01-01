package lexicalproviders

import (
	"path/filepath"
	"strings"
	"unicode"
)

// List created based on Product Spec heuristics and results from
// `local-pipelines/lexical/train/cmds/lineending` binary
var globalWhitelist = []string{"++", "--", ")", "]", "}", "'", "\"", ";"}
var whitelistByExt = map[string][]string{
	"c":     {":"},
	"cpp":   {">"},
	"cs":    {">"},
	"css":   {">"},
	"go":    {"..."},
	"h":     {">"},
	"hpp":   {">"},
	"java":  {">"},
	"js":    {">", "`"},
	"jsx":   {">", "`"},
	"kt":    {"?"},
	"php":   {">"},
	"py":    {"\\", ":", "..."},
	"rb":    {"?", "!", "|"},
	"scala": {">"},
	"sh":    {"\\"},
	"ts":    {">", "`", "*"},
	"tsx":   {">", "`"},
	"vue":   {">", "`"},
}

var globalBlacklist = []string{"+", "-", "*", "/", "%", "=", "~", "&", "^", "<", ".", "!", "|", ">", ":", ","}
var blacklistByExt = map[string][]string{
	"less": {"@"},
	"py":   {"$", "#"},
	"rb":   {"@", "#"},
	"sh":   {"$", "#"},
	"vue":  {"@"},
}

// HasValidSuffix ...
func HasValidSuffix(comp string, path string) bool {
	var ext string
	if e := filepath.Ext(path); len(e) > 0 {
		ext = e[1:]
	}

	// remove trailing whitespace
	comp = strings.TrimRightFunc(comp, unicode.IsSpace)

	// Allow letter endings
	if len(comp) > 0 {
		r := rune(comp[len(comp)-1])
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}

	// Check in the order of white list, black list
	for _, s := range globalWhitelist {
		if strings.HasSuffix(comp, s) {
			return true
		}
	}
	if wl, ok := whitelistByExt[ext]; ok {
		for _, s := range wl {
			if strings.HasSuffix(comp, s) {
				return true
			}
		}
	}

	for _, s := range globalBlacklist {
		if strings.HasSuffix(comp, s) {
			return false
		}
	}
	if bl, ok := blacklistByExt[ext]; ok {
		for _, s := range bl {
			if strings.HasSuffix(comp, s) {
				return false
			}
		}
	}
	return true
}
