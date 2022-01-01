package syntaxcolors

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"os"
	"time"
	"io/ioutil"
	"path"
	"path/filepath"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindata_file_info struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindata_file_info) Name() string {
	return fi.name
}
func (fi bindata_file_info) Size() int64 {
	return fi.size
}
func (fi bindata_file_info) Mode() os.FileMode {
	return fi.mode
}
func (fi bindata_file_info) ModTime() time.Time {
	return fi.modTime
}
func (fi bindata_file_info) IsDir() bool {
	return false
}
func (fi bindata_file_info) Sys() interface{} {
	return nil
}

var _syntaxcolors_css = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x74\x54\xcd\x6e\xf3\x28\x14\xdd\xe7\x29\xae\x34\xbb\xaa\x4e\x69\xf3\xd3\x34\xd1\x48\xd3\x45\x17\x23\xcd\x6e\x9e\x00\xc3\xb5\x83\x8c\xc1\x02\x92\x36\xaa\xfa\xee\xdf\x05\x17\xff\x24\xad\x23\xa1\xe0\x73\x0e\xdc\xbf\xe3\x87\x3b\x90\xe8\xd1\x05\xf0\xe2\x88\x2d\x42\x67\x5d\x40\x09\x95\xb3\x2d\x9c\x55\x0b\xc1\x42\x6d\x6d\xad\x09\x71\x18\x82\xaa\x2e\x70\xf7\xb0\xa0\xff\xcb\xb4\xbf\x74\x4e\x99\x00\x9f\x20\x95\xef\x34\xbf\xec\xa1\xd4\x56\x34\x07\x28\xb9\x68\x6a\x67\x4f\x46\x16\xc2\x6a\xeb\xf6\xf0\xd7\x6a\xb5\x82\xaf\x28\x85\xa5\xb1\xc2\x4a\x24\xd9\x2d\xcd\x58\x83\x07\xc8\x1a\xc6\x58\xd6\xf8\xe0\x48\x90\x81\xaa\xe2\x8c\x13\x06\x94\x01\x21\xca\xd4\x00\x05\x74\xca\x34\xdf\xf1\xc1\xb2\x79\x97\x53\x01\xc3\xed\x4e\xe4\xc3\x04\xa5\x37\x62\xbb\x67\x81\x58\xf6\x87\x11\xd2\x22\xa5\x54\x80\x6f\x2e\xa5\x3e\xe1\x70\x1e\x25\x3b\xd1\xbc\xec\xaa\xf2\x65\xd7\x6b\x08\x41\x80\x18\x80\x56\xf5\x31\xd4\x0e\xd1\x0c\x32\xad\xc2\x44\x26\xe4\x46\x6c\x44\x2f\x23\x04\x1d\xd7\x24\x93\xdc\x35\x8e\xca\x9e\x35\xdd\xc9\xcc\x72\xad\x48\x40\x0f\x69\x08\x11\xe1\xc4\x83\xb2\xe3\x0d\x9d\xfe\x95\xad\x39\xb5\x07\x3f\xc2\x98\x04\xaf\x7f\x28\x4a\xe4\x1e\x43\xab\x1f\x3e\x5a\x0d\x91\x32\x26\x73\x41\xad\xed\xfb\xa0\xe7\x61\x7a\x57\x29\xcb\xe7\xed\x77\xe1\x78\xa0\x36\x94\xa7\x80\x60\x38\x0d\x12\xe9\x9b\x23\x6f\xd4\x44\x79\xfe\xa5\x7f\xa3\xf2\xcc\x63\xc1\xaf\xfa\x28\x51\xfc\x52\x77\x42\x54\x4b\x05\xcc\xcf\x75\xfd\x17\xc4\xf9\xbf\x23\x12\x4d\xad\xd0\xdc\xfb\xbf\xb5\x32\x68\x4e\xad\x07\x2a\x1f\x8f\x23\x9d\x06\x1c\x03\x44\x00\x08\x29\x31\xcd\x12\x89\xad\x5e\x0e\xec\x4f\x68\xb9\xab\x95\x29\x82\xed\xf6\xc0\x0e\x79\x5b\xda\x10\x6c\x9b\xde\xe4\xf8\x5e\xdf\xe2\xaf\x8f\xef\xdf\x37\x50\x46\xd2\x34\x79\xf2\x12\xcf\x22\x8d\x55\xea\x87\x56\xcb\xff\xd8\x7d\x5c\x1f\xd3\xfa\x94\xd6\x55\x5a\x37\x69\xdd\xa6\xf5\x39\xad\x3b\x0a\x42\x2b\x1f\x0a\x1f\x2e\x1a\x8b\x38\x72\xbd\x59\x68\xa4\xe9\xaa\x57\x4d\xb3\x64\x38\xd5\xd0\x1f\xb9\x8c\x29\x54\xd6\xa5\xac\x7c\xbe\xeb\xf1\xe6\xfc\xfe\xe4\x17\x3a\xf9\x6b\xb1\xf8\xa7\x45\xa9\x62\x4d\x92\xa1\x17\x00\xb7\x26\xff\xd9\xad\x24\x4e\xe4\x64\xd1\x7b\x48\xce\xbe\x72\x2b\xdb\xb2\x91\x45\xbe\xcc\xac\xb9\x45\x19\xdb\x8e\x2c\xf2\x61\x66\xcd\xcd\xba\x65\x93\xb3\x28\xb6\xcc\x9a\xdb\x73\xcd\xd6\x23\x8b\x8c\x96\x59\x73\x37\xb2\xf5\x84\x45\xd6\xca\xac\xb9\xff\xd6\xeb\xc9\x8d\x64\xb6\x81\x35\xf3\x1d\x9b\xc5\xc5\xeb\x21\xae\x99\xe3\x66\x39\x92\x99\x32\x6b\xee\xab\x59\xf4\x64\x9c\x91\x75\xbe\xa9\x2a\xb5\x6e\x29\x4e\xce\x53\xbb\x63\xd7\x6e\x7b\x44\x1f\x96\x03\x01\xc3\x17\x5a\x99\x38\x16\x45\xff\xa1\x26\xe0\x5d\xc9\x70\xdc\xc3\xba\xfb\x88\xbb\x23\x46\x0b\xed\xe1\x89\xf5\xfb\xab\x49\x2f\x36\xf1\xf5\xd7\xe2\x4f\x00\x00\x00\xff\xff\xa5\x8b\xda\xdf\x39\x06\x00\x00")

func syntaxcolors_css_bytes() ([]byte, error) {
	return bindata_read(
		_syntaxcolors_css,
		"syntaxcolors.css",
	)
}

func syntaxcolors_css() (*asset, error) {
	bytes, err := syntaxcolors_css_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "syntaxcolors.css", size: 1593, mode: os.FileMode(420), modTime: time.Unix(1430526148, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if (err != nil) {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"syntaxcolors.css": syntaxcolors_css,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() (*asset, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"syntaxcolors.css": &_bintree_t{syntaxcolors_css, map[string]*_bintree_t{
	}},
}}

// Restore an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, path.Dir(name)), os.FileMode(0755))
        if err != nil {
                return err
        }
        err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
        if err != nil {
                return err
        }
        err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
        if err != nil {
                return err
        }
        return nil
}

// Restore assets under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        if err != nil { // File
                return RestoreAsset(dir, name)
        } else { // Dir
                for _, child := range children {
                        err = RestoreAssets(dir, path.Join(name, child))
                        if err != nil {
                                return err
                        }
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

