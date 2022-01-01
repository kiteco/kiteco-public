package render

import (
	"bytes"
	"regexp"

	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"gopkg.in/yaml.v1"
)

// FrontMatter is the schema of front matter before the --- line
type FrontMatter struct {
	Slugs        []string                  `yaml:"slugs,omitempty"`
	Environments map[string]execution.Spec `yaml:"environments,omitempty"`
	Headers      map[string]string         `yaml:"headers,omitempty"`
}

var frontMatterBoundaryRE = regexp.MustCompile(`(?m)^---\r?$`)

func splitFrontMatter(src []byte) (FrontMatter, []byte, error) {
	idxs := frontMatterBoundaryRE.FindIndex(src)
	if idxs == nil {
		return FrontMatter{}, src, nil
	}

	var front FrontMatter
	if err := yaml.Unmarshal(src[:idxs[0]], &front); err != nil {
		return FrontMatter{}, nil, err
	}

	return front, src[idxs[1]:], nil
}

// Raw represents a raw post
type Raw struct {
	// FrontMatter can be manipulated from the outside by tooling
	FrontMatter

	title    string
	markdown []byte
	after    []byte
}

// Encode re-encodes the pre-processed post data as text: it is the inverse of Preprocess
func (r Raw) Encode() ([]byte, error) {
	yaml, err := yaml.Marshal(r.FrontMatter)
	if err != nil {
		return nil, err
	}
	// no newline before or after --- to make Join the inverse of Split
	// yaml already has a trailing newline
	return bytes.Join([][]byte{yaml, []byte("---"), r.markdown}, []byte{}), nil
}

// Title gets the raw post title
func (r Raw) Title() string {
	return r.title
}

// ParseRaw preprocesses post text data into an in-memory struct
func ParseRaw(src []byte) (Raw, error) {
	front, md, err := splitFrontMatter(src)
	if err != nil {
		return Raw{}, err
	}
	headline, after := splitOnHeadline(md)
	title := string(bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimSpace(headline), []byte("#"))))

	return Raw{
		FrontMatter: front,

		title:    title,
		markdown: md,
		after:    after,
	}, nil
}
