package handlers

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

func kindForHint(hint string) types.CompletionItemKind {
	switch hint {
	case "call", "function":
		return types.FunctionCompletion
	case "descriptor":
		return types.PropertyCompletion
	case "keyword":
		return types.KeywordCompletion
	case "module":
		return types.ModuleCompletion
	case "snippet":
		return types.SnippetCompletion
	case "type":
		return types.ClassCompletion
	case "unknown":
		return types.TextCompletion
	default:
		return types.ValueCompletion
	}
}

var errNotAbsolute = errors.New("path is not absolute")

// https://golang.org/src/cmd/go/internal/web/url.go
func filepathFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", errors.New("non-file URL")
	}

	if u.Path == "" {
		if u.Host != "" || u.Opaque == "" {
			return "", errors.New("file URL missing path")
		}
		return filepath.FromSlash(u.Opaque), nil
	}

	path, err := convertFileURLPath(u.Host, u.Path)
	if err != nil {
		return path, err
	}
	return path, nil
}

func utf8OffFromPos(text string, pos types.Position) (int, error) {
	var off int

	// go to the right line
	for l := 0; l < pos.Line; l++ {
		i := strings.IndexByte(text, '\n')
		if i < 0 {
			return off, errors.New("line number out of bounds")
		}
		off += i + 1
		text = text[i+1:]
	}

	// extract the current line
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	// strip possible CR ending from CR-LF
	if len(text) > 0 && text[len(text)-1] == '\r' {
		text = text[:len(text)-1]
	}

	coff, err := stringindex.NewConverter(text).EncodeOffset(
		pos.Character, stringindex.UTF16, stringindex.UTF8)
	if err != nil {
		return off, errors.Wrapf(err, "character offset out of bounds of line")
	}

	return off + coff, nil
}

func posFromUTF8Off(text string, off int) (types.Position, error) {
	var pos types.Position

	if off > len(text) {
		return pos, errors.New("offset out of bounds")
	}

	for i := 0; i < off; i++ {
		if text[i] == '\n' {
			pos.Line++
			pos.Character = 0
		} else {
			pos.Character++
		}
	}
	// now pos.Character is a UTF8 offset
	coff, err := stringindex.NewConverter(text[off-pos.Character:off]).EncodeOffset(
		pos.Character, stringindex.UTF8, stringindex.UTF16)
	if err != nil {
		panic("this should be impossible")
	}

	pos.Character = coff

	return pos, nil
}

func buildURL(base string, params map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func copyFile(src, dst string) (int64, error) {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	if cleanSrc == cleanDst {
		return 0, nil
	}
	sf, err := os.Open(cleanSrc)
	if err != nil {
		return 0, err
	}
	defer sf.Close()
	if err := os.Remove(cleanDst); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	df, err := os.Create(cleanDst)
	if err != nil {
		return 0, err
	}
	defer df.Close()
	return io.Copy(df, sf)
}
