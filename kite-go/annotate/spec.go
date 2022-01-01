package annotate

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	yaml "gopkg.in/yaml.v1"
)

// This package allows curators to customize time and output limits for code examples, but
// we need to specify "meta-limits" beyond which the per-example limits cannot be extended.

const (
	maxByteLimit  = 1000000
	maxLineLimit  = 10000
	maxTimeout    = 60 * time.Second
	defaultSaveAs = "src.py"
)

var (
	// SamplesDir is the S3 path that contains sample files to be used by
	// curated examples.
	SamplesDir = fileutil.Join("s3://kite-data", SamplesBase)

	// SamplesBase is the name of the directory that contains sample files.
	SamplesBase = "sample-files"

	// httpInternalPort is the port used for HTTP communication. 8000 was chosen because
	// it is above 1024 (so it does not require root) and is widely used as a port for
	// "in-development" servers.
	httpInternalPort = 8000
)

type request struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Body    string            `yaml:"body"`
	Form    map[string]string `yaml:"form"`
	Headers map[string]string `yaml:"headers"`
	Cookies map[string]string `yaml:"cookies"`
	Files   []file            `yaml:"files"`
}

type file struct {
	Field string `yaml:"field"`
	Path  string `yaml:"path"`
	Data  string `yaml:"data"`
}

// InputFile is a file with user-specified data (through postlude spec)
// or a sample file (from S3). If it's a directory on S3, the `files_to_show`
// field can be used to identify specific files to be shown from the directory.
type InputFile struct {
	Name           string   `yaml:"name"`
	Location       string   `yaml:"location"`
	Contents       string   `yaml:"contents"`
	ContentsBase64 string   `yaml:"contents_base64"`
	Hide           bool     `yaml:"hide"`
	FilesToShow    []string `yaml:"files_to_show"`
}

// Spec is embedded in code examples as a yaml string
type Spec struct {
	Stdin       string            `yaml:"stdin"`        // string to send on standard input stream
	Args        []string          `yaml:"args"`         // command line arguments
	Env         map[string]string `yaml:"env"`          // environment variables
	Limits      *sandbox.Limits   `yaml:"limits"`       // time and output limits
	HTTPRequest *request          `yaml:"http_request"` // path for http request
	SampleFiles []string          `yaml:"sample_files"` // sample files (backwards compatibility)
	InputFiles  []*InputFile      `yaml:"input_files"`  // array of input files
	Inline      bool              `yaml:"inline"`       // true or false
	SaveAs      string            `yaml:"save_as"`      // file name of main source
	Command     string            `yaml:"command"`      // main command to run in container
}

// ParseSpec constructs an Spec struct out of the given spec string.
// If the format is different, it returns an error.
func ParseSpec(s string) (*Spec, error) {
	spec := Spec{
		Inline: true,
		// In the future we might switch on language & set different defaults
		// for different languages.
		Limits: sandbox.DefaultLimits,
		SaveAs: defaultSaveAs,
		Env:    make(map[string]string),
	}
	if s == "" {
		return &spec, nil
	}
	err := yaml.Unmarshal([]byte(s), &spec)
	if err != nil {
		return nil, err
	}

	// This makes processing later a lot easier:
	if spec.HTTPRequest != nil {
		if spec.HTTPRequest.Headers == nil {
			spec.HTTPRequest.Headers = make(map[string]string)
		}
		if spec.HTTPRequest.Form == nil {
			spec.HTTPRequest.Form = make(map[string]string)
		}
	}

	return &spec, nil
}

// NewSpecFromCode extracts and parses a Spec structure out of the given code string.
func NewSpecFromCode(language lang.Language, code string) (*Spec, error) {
	specStr, err := ExtractSpec(language, code)
	if err != nil {
		return nil, err
	}
	spec, err := ParseSpec(specStr)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// ExtractSpec an apparatus spec from given code and language
func ExtractSpec(language lang.Language, code string) (string, error) {
	switch language {
	case lang.Bash:
		return "", nil // no specs are supported in bash
	case lang.Python:
		return extractPythonSpec(code), nil
	}
	return "", fmt.Errorf("cannot extract apparatus spec from %s", language.Name())
}

func extractPythonSpec(code string) string {
	delim := "'''"
	beginpos := strings.Index(code, delim)
	endpos := strings.LastIndex(code, delim)
	if beginpos == -1 || endpos == -1 || beginpos == endpos {
		return ""
	}
	// Begins on new line
	if beginpos > 0 && code[beginpos-1] != '\n' {
		return ""
	}
	// At ending of code
	if endpos+len(delim) != len(strings.TrimRight(code, "\n")) {
		return ""
	}
	return code[beginpos+len(delim) : endpos]
}

// RemoveSpecFromCode removes a Spec structure out of the given code string.
func RemoveSpecFromCode(language lang.Language, code string) string {
	switch language {
	case lang.Python:
		return removePythonSpec(code)
	case lang.Bash:
		return code // no specs are supported in bash
	default:
		return code
	}
}

// removePythonSpec returns source with the spec removed. If there is no spec then it returns
// the input string.
func removePythonSpec(code string) string {
	delim := "'''"
	beginpos := strings.Index(code, delim)
	endpos := strings.LastIndex(code, delim)
	if beginpos == -1 || endpos == -1 || beginpos == endpos {
		return code
	}
	// Check that the begin position is at the start of a line
	if beginpos > 0 && code[beginpos-1] != '\n' {
		return code
	}
	if endpos+len(delim) != len(strings.TrimRight(code, "\n")) {
		return code
	}
	return code[:beginpos] + code[endpos+len(delim):]
}

// Validates a spec and returns a list of messages, each describing an issue.
// If the returned list is empty then the spec is valid.
func validateSpec(spec *Spec) []string {
	var issues []string
	if spec.Limits != nil {
		if spec.Limits.MaxBytes < 0 || spec.Limits.MaxBytes > maxByteLimit {
			issues = append(issues, fmt.Sprintf("byte limit out of range: %d", spec.Limits.MaxBytes))
		}
		if spec.Limits.MaxLines < 0 || spec.Limits.MaxLines > maxLineLimit {
			issues = append(issues, fmt.Sprintf("line limit out of range: %d", spec.Limits.MaxLines))
		}
		if spec.Limits.Timeout < 0 || spec.Limits.Timeout > maxTimeout {
			issues = append(issues, fmt.Sprintf("time limit out of range: %d", spec.Limits.Timeout))
		}
	}
	if spec.HTTPRequest != nil {
		if spec.HTTPRequest.URL == "" {
			issues = append(issues, "a URL is required")
		}
		if spec.HTTPRequest.Body != "" && len(spec.HTTPRequest.Form) > 0 {
			issues = append(issues, fmt.Sprintf("body and form cannot both be specified"))
		}
		if len(spec.HTTPRequest.Files) > 0 && len(spec.HTTPRequest.Body) > 0 {
			issues = append(issues, fmt.Sprintf("files and form cannot both be specified"))
		}
	}
	for i, f := range spec.InputFiles {
		var count int
		if f.Location != "" {
			count++
		}
		if f.Contents != "" {
			count++
		}
		if f.ContentsBase64 != "" {
			count++
		}
		if count != 1 {
			issues = append(issues, fmt.Sprintf("specify exactly one of location, contents or contents_base64"))
		}
		if (f.Contents != "" || f.ContentsBase64 != "") && f.Name == "" {
			issues = append(issues, fmt.Sprintf("please specify name for input file %d\n", i))
		}
		if f.Hide && len(f.FilesToShow) > 0 {
			issues = append(issues, "please specify only one of hide or files_to_show, not both")
		}
	}
	return issues
}

// BuildApparatus creates an apparatus from a Spec structure.
func (spec *Spec) BuildApparatus() (*sandbox.Apparatus, error) {
	issues := validateSpec(spec)
	if len(issues) > 0 {
		return nil, errors.New(strings.Join(issues, ", "))
	}

	// Process limits
	var limits *sandbox.Limits
	if spec.Limits != nil {
		limits = spec.Limits
	} else {
		limits = sandbox.DefaultLimits
	}

	// Process HTTP request
	var port int
	var action sandbox.Action
	if spec.HTTPRequest != nil {
		// Deal with the body
		body := &bytes.Buffer{}
		if len(spec.HTTPRequest.Files) > 0 {
			// Create a multipart body
			w := multipart.NewWriter(body)
			for _, f := range spec.HTTPRequest.Files {
				part, err := w.CreateFormFile(f.Field, f.Path)
				if err != nil {
					return nil, fmt.Errorf("error writing %s to form data: %v", f.Path, err)
				}
				_, err = part.Write([]byte(f.Data))
				if err != nil {
					return nil, fmt.Errorf("error writing %s to form data: %v", f.Path, err)
				}
			}
			for k, v := range spec.HTTPRequest.Form {
				w.WriteField(k, v)
			}
			w.Close()
			spec.HTTPRequest.Headers["Content-Type"] = "multipart/form-data; boundary=" + w.Boundary()
		} else if len(spec.HTTPRequest.Form) > 0 {
			// Create a urlencoded body
			form := url.Values{}
			for k, v := range spec.HTTPRequest.Form {
				form.Set(k, v)
			}
			body.WriteString(form.Encode())
			spec.HTTPRequest.Headers["Content-Type"] = "application/x-www-form-urlencoded"
		} else if spec.HTTPRequest.Body != "" {
			// Create a raw string body
			body.WriteString(spec.HTTPRequest.Body)
		}

		if body != nil && body.Len() > 0 {
			spec.HTTPRequest.Headers["Content-Length"] = strconv.Itoa(body.Len())
		}

		// Construct the request
		request, err := http.NewRequest(spec.HTTPRequest.Method, spec.HTTPRequest.URL, body)
		if err != nil {
			return nil, err
		}
		for k, v := range spec.HTTPRequest.Headers {
			request.Header.Set(k, v)
		}
		for k, v := range spec.HTTPRequest.Cookies {
			request.AddCookie(&http.Cookie{
				Name:  k,
				Value: v,
			})
		}
		// Construct the apparatus
		action = &sandbox.DoThenCancel{
			Action: &sandbox.HTTPAction{
				Request: request}}
		port = httpInternalPort
	}

	apparatus := sandbox.NewApparatus(action, limits, port, spec.Command)

	// For backwards compatibility, while the curators change all examples
	// that use `sample_files` to use the new `input_files` instead.
	// Will be removed immediately after.
	for _, name := range spec.SampleFiles {
		path := SamplesDir + "/" + name
		data, err := fileutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("unable to read %s (at %s): %v", name, path, err)
		}
		apparatus.AddFile(name, data)
	}

	for _, f := range spec.InputFiles {
		var data []byte
		if f.Location != "" {
			if err := fetchAndAddSampleFiles(f, apparatus); err != nil {
				return nil, err
			}
			continue
		} else if f.Contents != "" {
			data = []byte(f.Contents)
		} else if f.ContentsBase64 != "" {
			var err error
			data, err = base64.StdEncoding.DecodeString(f.ContentsBase64)
			if err != nil {
				return nil, fmt.Errorf("error decoding base64 string for file %s: %v", f.Name, err)
			}
		}

		apparatus.AddFile(f.Name, data)
	}

	return apparatus, nil
}

func fetchAndAddSampleFiles(f *InputFile, apparatus *sandbox.Apparatus) error {
	path := fileutil.Join(SamplesDir, f.Location)
	s3url, err := awsutil.ValidateURI(path)
	if err != nil {
		return err
	}
	bucket, err := awsutil.GetBucket(s3url.Host)
	if err != nil {
		return err
	}
	key := strings.TrimPrefix(s3url.Path, "/")

	resp, err := bucket.List(key, "", "", 1000)
	if err != nil {
		return fmt.Errorf("error listing bucket: %+v", err)
	}
	for _, k := range resp.Contents {
		data, err := bucket.Get(k.Key)
		if err != nil {
			return fmt.Errorf("error getting key %s from bucket: %+v", k.Key, err)
		}
		k.Key = strings.TrimPrefix(k.Key, SamplesBase+"/")
		if k.Key != f.Location {
			k.Key = strings.TrimPrefix(k.Key, f.Location+"/")
		}
		if !apparatus.FileExists(k.Key) {
			apparatus.AddFile(k.Key, data)
		}
	}
	return nil
}

// NewApparatusFromCode parses a apparatus spec from the raw code example source code and
// returns an apparatus that executes that spec. If there is not spec then this function
// returns a default apparatus. If there is a spec an it fails to parse then this function
// returns an error.
func NewApparatusFromCode(code string, language lang.Language) (*sandbox.Apparatus, error) {
	spec, err := NewSpecFromCode(language, code)
	if err != nil {
		return nil, err
	}

	return spec.BuildApparatus()
}
