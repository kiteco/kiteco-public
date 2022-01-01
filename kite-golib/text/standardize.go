package text

import (
	"github.com/djimenez/iconv-go"
	"github.com/gogs/chardet"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// StandardizeEncoding of s to utf8
func StandardizeEncoding(s string) (string, error) {
	encs, err := chardet.NewTextDetector().DetectAll([]byte(s))
	if err != nil {
		return "", errors.Errorf("error detecting encoding: %v", err)
	}

	for _, enc := range encs {
		out, err := iconv.ConvertString(s, enc.Charset, "utf-8")
		if err == nil {
			return out, nil
		}
	}
	return "", errors.Errorf("unable to convert string to utf8")
}
