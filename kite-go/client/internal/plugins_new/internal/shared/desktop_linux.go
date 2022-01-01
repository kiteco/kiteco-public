package shared

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// DesktopFile represents the structure of a Linux XDG desktop file
type DesktopFile struct {
	path string
	data map[string]string
}

// ReadDesktopFile parses the file as a desktop file and returns it.
// If the parsing failed then an error is returned
// The file spec is at https://standards.freedesktop.org/desktop-entry-spec/latest/ar01s06.html
func ReadDesktopFile(file string) (*DesktopFile, error) {
	// read all of the file
	in, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	bytes, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	// split into separate lines
	lines := strings.Split(string(bytes), "\n")
	// ignore all lines before the main section
	for len(lines) > 0 {
		first := lines[0]
		lines = lines[1:]
		if first == "[Desktop Entry]" {
			break
		}
	}

	// read all keys and values until the next section or EOF is reached
	// ignore line comments
	data := make(map[string]string)
	for len(lines) > 0 {
		first := lines[0]
		lines = lines[1:]

		if strings.HasPrefix(first, "#") {
			// ignore line comment
			continue
		}

		if strings.HasPrefix(first, "[") {
			// new section, stop
			break
		}

		eqIndex := strings.Index(first, "=")
		if eqIndex == -1 {
			// ignore invalid lines
			continue
		}

		key := first[0:eqIndex]
		value := first[eqIndex+1:]
		data[key] = value
	}

	return &DesktopFile{
		path: file,
		data: data,
	}, nil
}

// Path returns the filepath of this desktop file
func (d *DesktopFile) Path() string {
	return d.path
}

// Value returns the value for the given key, if present
// an error is returned if no value for the key is available
func (d *DesktopFile) Value(key string) (string, error) {
	v, ok := d.data[key]
	if !ok {
		return "", errors.New("unknown key")
	}
	return v, nil
}

// ExecPath returns the path to the executed binary with argument and argument placeholders removed
// It first checks the value of TryExec and then the value of Exec
func (d *DesktopFile) ExecPath() (string, error) {
	// the TryExec is a reference to the binary without flags
	v, err := d.Value("TryExec")
	if err == nil {
		return v, nil
	}

	// next is Exec, which contains an optionally quoted path and flags
	v, err = d.Value("Exec")
	if err == nil && len(v) > 0 {
		// if quoted take quoted part as path name
		if v[0] == '"' {
			// skip initial quote
			v = v[1:]
			end := strings.Index(v, "\"")
			if end != -1 {
				return v[:end], nil
			}
		}

		// if unquoted take the part before the first whitespace
		end := strings.Index(v, " ")
		if end != -1 {
			return v[:end], nil
		}

		// fallback to full value if without quotes and whitespace
		return v, nil
	}

	return "", errors.New("no path found")
}

// CollectDesktopFiles reads and collects all desktop files in a given directory
func CollectDesktopFiles(dir string) ([]*DesktopFile, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.desktop"))
	if err != nil {
		return nil, err
	}

	var result []*DesktopFile
	for _, f := range files {
		d, err := ReadDesktopFile(f)
		if err != nil {
			continue
		}

		result = append(result, d)
	}
	return result, err
}
