// Code generated by go-bindata.
// sources:
// templates/cooccurences.html
// templates/toplevel.html
// DO NOT EDIT!

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
	"path/filepath"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _templatesCooccurencesHtml = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x9c\x55\xcd\x6e\xdb\x30\x0c\xbe\xef\x29\x38\xa3\xd7\x48\x5b\xb3\x5d\x32\xd9\x97\x1e\x8a\x02\x43\x31\x6c\xd8\x03\xc8\x92\x52\x3b\xb5\x25\x4f\x52\x8a\x04\x9e\xdf\x7d\x94\x1d\xff\xc2\xeb\xb2\xe6\x10\x8b\xa4\xc8\xef\xe3\x9f\xcd\xde\x4b\x23\xfc\xb9\x52\x90\xf9\xb2\x48\xde\xb1\xee\x01\xf8\x63\x99\xe2\xb2\x3b\xb6\xa2\xcf\x7d\xa1\x92\xfb\xdc\x67\xc7\x14\xbe\x71\xf1\xcc\x9f\x14\xfc\xf0\xdc\x3b\x46\x3b\xdb\x78\xd7\x09\x9b\x57\x1e\x42\xe0\x38\xf2\xea\xe4\xe9\x81\xbf\xf0\x4e\x1b\x81\xb3\x22\x8e\x32\xef\x2b\xb7\xa3\x54\x18\xa9\xc8\xe1\xd7\x51\xd9\x33\x11\xa6\xa4\xdd\x71\x73\x4b\x3e\x92\x4f\xa4\xcc\x35\x39\xb8\x28\x61\xb4\xf3\x7d\x1b\x44\xc9\x4f\x42\x6a\x92\x1a\xe3\x9d\xb7\xbc\x0a\x42\x80\x1a\x14\x74\x4b\xb6\xe4\x33\x3d\xb8\x51\xf5\x1a\x74\x91\xeb\x67\xb0\xaa\x88\x23\xe7\xcf\x85\x72\x99\x52\x08\x39\x61\x22\x9c\x8b\x20\xb3\x6a\xff\xff\x1c\xd0\x75\x41\x22\x04\x43\x16\x01\x74\x9a\x7e\x40\x1e\xe5\xd4\xc8\x33\xd4\x83\x18\x7e\x15\x97\x32\xd7\x4f\x3b\xb8\xfd\x50\x9d\xbe\x0c\xa6\x66\x38\x79\xb9\xf0\xd8\x1b\xed\x37\x7b\x5e\xe6\xc5\x79\x07\xa5\xd1\xc6\x55\x5c\xa8\x35\x57\x2c\xca\x88\xcf\xe8\x38\x29\x2c\x10\xe9\x69\x31\x99\xbf\x80\x28\xb8\x73\x58\x17\x9e\x16\x2a\x9a\x8e\x53\x50\xcc\xac\xd0\xfe\x6f\x30\xf3\xbc\x52\x72\x72\x37\xdc\x9e\xc6\xed\x34\x36\x99\x91\x0f\x2a\x09\xc2\x14\x48\x5a\xc7\xd1\x16\x6b\x96\x26\x75\x4d\x2e\x93\xda\x34\x8c\xa6\x58\x46\x2f\xaf\x72\xcb\xf7\x40\x1e\xf4\x3d\xf6\x20\x6b\x9a\x07\x0d\xed\xa9\xae\x55\xe1\x30\xd2\xa3\xf1\x30\xd1\x69\x79\x7d\xf0\x45\x64\xc6\x2f\x63\x82\x44\x1f\x71\x11\xe0\x37\x68\x7c\xfc\xfc\xfe\xb5\x69\xa2\x05\x7b\x9e\x0c\x58\x73\x1c\x94\xed\xab\xa5\x59\xc9\xf0\xce\x18\x21\x8e\x16\xe7\xa3\x5f\xe5\xf5\x0c\x56\x5c\x5b\xea\x57\xdf\xbe\x00\x29\x2d\x14\xdc\x99\xa3\xf6\x6b\x9e\x6d\x06\x00\x13\x4d\x5d\x5b\xae\xf1\xfd\x72\x23\x60\x17\x03\x99\x44\x71\x4d\xb3\xc0\xfc\xf7\x20\x70\x68\xc7\x35\x8e\x50\x6b\xec\xae\xae\x6f\x44\x5f\xee\x56\x83\xc5\x1e\xfa\x80\xa6\xfe\xe5\x16\xac\x23\xee\xd0\x94\xf1\x46\xd7\x96\xab\xfb\x8e\x8e\x6b\x9d\x1f\xc9\xcc\x7a\xbf\x84\xf9\x4b\xf7\xd7\xd1\xd0\xb9\xad\x76\x9b\x43\x59\xf2\x35\xbf\xe5\xdc\x5c\x00\x26\x4b\x3e\xdb\x3a\x14\xc3\x7e\xf6\x3b\xdf\x9b\x18\xc5\x2d\xc7\x0f\x07\xed\xbe\x1c\x7f\x02\x00\x00\xff\xff\xef\x18\x41\x3b\x51\x06\x00\x00")

func templatesCooccurencesHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesCooccurencesHtml,
		"templates/cooccurences.html",
	)
}

func templatesCooccurencesHtml() (*asset, error) {
	bytes, err := templatesCooccurencesHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/cooccurences.html", size: 1617, mode: os.FileMode(420), modTime: time.Unix(1468457075, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _templatesToplevelHtml = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x9c\x54\xc1\x8e\xdb\x20\x10\xbd\xef\x57\x4c\xad\xbd\x06\xda\x4d\x7b\x71\xb1\x2f\x3d\x54\x95\xaa\x6a\x55\xa9\x1f\x40\x80\xac\xc9\xda\xe0\x02\xd9\xc6\x72\xfd\xef\x1d\xec\x38\x26\xe9\xae\xb2\xad\x0f\x06\xde\xbc\xf1\x7b\x0c\x83\xd9\x1b\x69\x45\xe8\x5a\x05\x55\x68\xea\xf2\x86\x4d\x03\xe0\xc3\x2a\xc5\xe5\x34\xc5\x45\xd0\xa1\x56\xe5\x3d\x17\x8f\xfc\x41\xc1\x27\x6b\x85\xd8\x3b\x65\x84\xf2\x8c\x4e\xb1\x99\xe9\x85\xd3\x6d\x80\xf8\xd1\x22\x0b\xea\x10\xe8\x8e\x3f\xf1\x09\xcd\xc0\x3b\x51\x64\x55\x08\xad\xcf\x29\x15\x56\x2a\xb2\xfb\xb9\x57\xae\x23\xc2\x36\x74\x9a\xae\xee\xc8\x3b\xf2\x9e\x34\xda\x90\x9d\xcf\x4a\x46\xa7\xdc\xff\x11\x68\xf8\x41\x48\x43\x36\xd6\x06\x1f\x1c\x6f\xe3\x22\x0a\x9d\x00\xba\x26\x6b\xf2\x81\xee\xfc\x02\xbd\x2c\x5c\x6b\xf3\x08\x4e\xd5\x45\xe6\x43\x57\x2b\x5f\x29\x85\x82\x89\x0f\xe1\x7d\x06\x95\x53\xdb\x7f\x77\x80\xa9\x17\x16\xe2\xc7\xd0\x43\x14\x5d\xb6\x1e\x75\xe7\xd5\xc6\xca\x0e\xfa\xe3\x02\xa0\xe5\x52\x6a\xf3\x90\xc3\xdd\xdb\xf6\xf0\xf1\x08\x0f\xc7\x91\x78\xc5\x9d\xa8\x56\xce\xfe\x4a\x52\x1a\xee\x1e\xb4\x59\x6d\x6c\x08\xb6\xc9\x61\xfd\x4c\x62\x90\x09\x7f\x6b\x4d\x58\x6d\x79\xa3\xeb\x2e\x87\xc6\x1a\xeb\x5b\x2e\xd4\x65\x0a\xd6\x6d\xb1\xc9\xe8\xd2\x46\x2c\x3a\x3e\xed\x45\xea\x27\x10\x35\xf7\x1e\x8b\xc7\x37\xb5\xca\xca\x93\x0e\x1b\x81\xb3\x28\x8c\xef\x15\x96\x47\xb7\x4a\x26\xdc\xc8\x4e\x3b\x75\xc6\xdc\x39\x30\xd2\x40\xd8\x1a\x2d\x9b\x22\x5b\x67\xe5\x45\x0f\x57\xd7\xf8\x9f\xf1\x60\xaa\xd7\x10\xef\x6d\xbb\xaf\xb9\xd3\xa1\xfb\x9b\x8d\xc8\x99\xb1\xc8\xb8\x30\xcf\x42\x5a\xa6\xf8\xf4\xbd\xe3\x06\x6f\xdd\xad\x80\xbc\x00\x92\x1a\x1f\x86\xeb\xdb\x96\x67\xee\x18\x87\xf1\x78\x8a\x0c\x51\xeb\xf2\xbe\xbf\x15\xe4\x1b\x5e\x44\xf8\x0d\x23\x32\x0c\x73\x0f\x8f\xa1\xf9\xca\xc7\xe8\xa2\xfb\xe3\xfb\x57\xe4\x95\x29\x63\x18\x18\xe5\xd8\xb0\x41\x5e\xb3\xd0\xf7\x7a\x8b\x9b\x21\x5f\xcc\x58\x53\x4c\xe4\xa9\xe2\xd1\x8c\xc1\xe1\x45\x99\xbe\x57\x46\xc6\xe9\x6b\xd4\x62\xf2\xe9\x4c\xc6\x8d\x34\x0d\x7f\x2e\xf9\xf2\x74\x8e\x2a\x37\x49\xfc\xac\x85\xe9\xd8\x94\x73\xa3\x2f\x21\x46\xb1\xb9\xf1\x67\x4a\xa7\xbf\xe9\x9f\x00\x00\x00\xff\xff\x6c\x81\x06\x00\x65\x05\x00\x00")

func templatesToplevelHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesToplevelHtml,
		"templates/toplevel.html",
	)
}

func templatesToplevelHtml() (*asset, error) {
	bytes, err := templatesToplevelHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/toplevel.html", size: 1381, mode: os.FileMode(420), modTime: time.Unix(1468457034, 0)}
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
	"templates/cooccurences.html": templatesCooccurencesHtml,
	"templates/toplevel.html": templatesToplevelHtml,
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
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"templates": &bintree{nil, map[string]*bintree{
		"cooccurences.html": &bintree{templatesCooccurencesHtml, map[string]*bintree{
		}},
		"toplevel.html": &bintree{templatesToplevelHtml, map[string]*bintree{
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
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

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        // File
        if err != nil {
                return RestoreAsset(dir, name)
        }
        // Dir
        for _, child := range children {
                err = RestoreAssets(dir, filepath.Join(name, child))
                if err != nil {
                        return err
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

