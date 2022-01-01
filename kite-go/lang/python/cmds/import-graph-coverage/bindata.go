package main

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

var _templates_result_html = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xdc\x56\x4d\x6f\xdb\x38\x10\xbd\xef\xaf\x60\x08\x03\xb1\x81\xa5\x94\xcd\xc7\x45\x91\x7d\x59\x60\x81\xbd\x04\x45\xd1\x5b\xd1\x03\x2d\xd2\x12\x1d\x8a\x54\xc9\x71\x1c\xd7\xf0\x7f\xef\x50\x92\x6d\x49\x56\x02\xb4\x40\x81\xa2\x3a\x24\xe4\x70\x86\xef\xcd\x9b\xe1\xc0\xe9\x95\xb0\x19\xec\x2a\x49\x0a\x28\xf5\xe2\xaf\xb4\xf9\x47\xf0\x4b\x0b\xc9\x45\xb3\xac\xb7\xa0\x40\xcb\xc5\xff\x65\x65\x1d\x90\xdc\xf1\xaa\x20\x99\x7d\x91\x8e\xe7\x32\x8d\x9b\xc3\xb3\xb3\xcf\x9c\xaa\x80\x84\x9b\xe7\x14\xe4\x2b\xc4\x6b\xfe\xc2\x1b\x2b\x25\xde\x65\x73\x5a\x00\x54\x3e\x89\xe3\xcc\x0a\x19\xad\xbf\x6e\xa4\xdb\x45\x99\x2d\xe3\x66\xc9\x6e\xa3\x7f\xa2\xfb\xa8\x54\x26\x5a\x7b\xba\x48\xe3\x26\xf6\xe7\x20\x4a\xfe\x9a\x09\x13\x2d\xad\x05\x0f\xc8\x3c\x6c\x02\xd4\xc9\x10\xdf\x45\x77\xd1\x43\xbc\xf6\x67\xd3\x7b\xd0\x5a\x99\x67\xe2\xa4\x9e\x53\x0f\x3b\x2d\x7d\x21\x25\x42\x76\x98\x64\xde\x53\x52\x38\xb9\xfa\x71\x0e\x18\x3a\x20\x11\x2e\x43\x16\x01\xb4\x9b\x7e\x40\x3e\xef\x97\x56\xec\xc8\xfe\xb4\x0d\x5f\xc5\x85\x50\x26\x4f\xc8\xed\x4d\xf5\xfa\x78\x3a\x3a\x9c\x56\x41\xf9\x41\x0c\xf0\x25\xf3\xea\x9b\x4c\xc8\xfd\x58\x04\x32\xde\x18\xe8\xc5\xac\xac\x01\xb6\x95\x2a\x2f\x20\x41\x16\x5a\x8c\xc6\x81\xb5\x1a\x54\xc5\x94\x31\xd2\x0d\x30\xb7\x85\x02\xc9\x7c\xc5\x33\x84\xad\x9c\x64\x5b\xcc\xfc\xb1\xe7\x82\xe2\xb1\xad\x12\x50\x24\xe4\xe1\xa6\x97\x4d\x2f\xd1\x9b\xbe\x3d\x94\x82\x71\xad\x72\x93\x10\x2d\x57\x30\xc6\x0c\xc4\x80\x4d\x9d\xcf\x8a\x97\x4a\xef\x12\x52\x5a\x63\x6b\x62\x63\xa1\xd8\x18\xfd\x1a\xa4\xc3\x46\x99\x4c\xf1\x75\x6d\x4a\x69\x60\x16\x39\x7c\x4d\xbb\xe9\x6a\x63\x32\x50\xd6\x4c\x67\x03\xd8\xc9\x94\x7e\x16\x1c\x38\x03\x9b\xe7\x5a\xce\xaf\x5b\xc5\xae\xbf\xd0\xd9\x51\xbd\xe9\xac\x43\x63\xd6\xe5\xd1\xc1\x4d\xe3\xf3\xbb\x4d\x43\x57\x1c\xe9\xa4\x42\xbd\x90\x4c\x73\xef\xb1\x49\xf9\x52\x4b\xda\x7d\xdc\xc1\xd0\x3b\x25\xf5\x5f\x86\x6d\xa8\x2a\x29\x3a\xbe\xc1\xbb\x3f\x1a\x1a\x9b\xeb\x1b\x6a\xb7\xc5\x07\x9e\x3d\x37\x03\xa2\x18\x3b\xfe\x0f\xdb\x49\xbc\x75\xf8\x64\x81\xbc\xe1\x80\x96\x1e\x5c\xf0\x18\x50\x4a\xa1\x9b\x7c\xf8\xf6\x7b\xc7\x4d\x2e\xc9\x44\xfd\x4d\x26\x4e\xfa\x8d\x06\x92\xcc\x49\xf4\xb1\x5e\xfa\xc3\xe1\xfd\x84\xd2\x2b\xc6\x82\xf5\xa8\xd2\xb1\x94\x94\xd4\x75\xb3\x4e\xe5\xca\x70\xcd\xea\x51\x38\xa7\x29\x76\xf2\x62\x65\x6d\x1a\x87\x05\x1d\xe4\x47\x9a\xa0\x30\x6d\x51\x6f\xb7\x91\xed\x2d\x19\x76\x1f\x57\xf8\x46\xe6\x34\xd0\x6f\xad\x6d\x4f\xd0\xb6\x0f\x5a\x6b\xa5\xb1\x31\x43\x73\x05\x5f\x00\x5b\xe2\x08\x6a\xb0\xe9\x82\x30\x76\x29\xa9\x58\xec\xf7\x6d\xe2\x51\x5b\x98\xc3\x01\xa5\x13\x23\xae\xc7\x2c\xeb\x07\x4f\x7f\x29\xd9\x71\xfd\x2e\x04\xab\x05\xbd\xb0\x76\xca\xba\xc6\xb2\x2a\x81\x10\xa1\xaa\xc7\x3c\x3f\xd9\xaa\xee\xa1\x41\x75\xcf\xc1\x4d\x48\xf4\xc4\x4b\x14\x23\x39\x1b\xfe\x0d\x89\x1f\x0e\x64\x14\x51\x8e\x5e\xd8\x94\x7a\x60\xa6\x1d\xd1\x5b\x26\x7f\xbe\xe4\xf8\x74\x7f\x1f\xd5\xcf\x64\x2e\x85\x1f\x0e\x92\x21\x08\x9e\xf7\x66\x68\x5c\x4f\xc5\xe3\xa4\x6d\x8e\x70\xe2\xd6\x3f\x9a\xbe\x07\x00\x00\xff\xff\x12\xce\xd8\x43\x4c\x09\x00\x00")

func templates_result_html_bytes() ([]byte, error) {
	return bindata_read(
		_templates_result_html,
		"templates/result.html",
	)
}

func templates_result_html() (*asset, error) {
	bytes, err := templates_result_html_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "templates/result.html", size: 2380, mode: os.FileMode(420), modTime: time.Unix(1445561383, 0)}
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
	"templates/result.html": templates_result_html,
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
	"templates": &_bintree_t{nil, map[string]*_bintree_t{
		"result.html": &_bintree_t{templates_result_html, map[string]*_bintree_t{
		}},
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

