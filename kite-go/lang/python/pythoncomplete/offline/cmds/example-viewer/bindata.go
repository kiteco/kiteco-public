// Code generated by go-bindata.
// sources:
// templates/ast.html
// templates/completions.html
// templates/example.html
// templates/examples.html
// templates/root.html
// DO NOT EDIT!

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	name    string
	size    int64
	mode    os.FileMode
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

var _templatesAstHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\x8e\xc1\xca\xc2\x30\x10\x84\xef\x7d\x8a\xa5\x94\x1e\x9b\xfb\xcf\x26\x3f\x1e\xbd\x09\xfa\x02\x91\x6c\x35\x10\xdb\xb0\xb5\x31\x10\xf6\xdd\x25\x81\xba\xa7\x65\xbe\x61\x66\x70\x0f\xa6\x03\x00\x28\x85\xed\xf2\x20\x18\x22\xaf\x09\xfe\x34\x4c\x17\x5e\x93\x77\xc4\x9b\x48\x73\xd4\xc3\xe0\x0d\x5a\x78\x32\xcd\xba\x57\x94\xed\x2b\x06\xfa\xf7\x2e\xeb\x52\x86\xe9\xec\xb2\xc8\x98\x3c\x7d\xb4\xdd\xde\xe3\x7d\x9f\x67\xe2\x4a\x6a\xa6\x48\x6f\x7e\x2f\x2a\x6b\x50\x05\x7f\x74\xd3\xe2\x44\x3a\x54\x75\x4d\x87\x91\xe9\x00\xd3\xe9\x7a\x6b\xa4\x69\xdf\x00\x00\x00\xff\xff\x9c\x98\x15\x3d\xae\x00\x00\x00")

func templatesAstHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesAstHtml,
		"templates/ast.html",
	)
}

func templatesAstHtml() (*asset, error) {
	bytes, err := templatesAstHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/ast.html", size: 174, mode: os.FileMode(436), modTime: time.Unix(1561486938, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesCompletionsHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x52\xdd\x6a\xdc\x3c\x10\xbd\xdf\xa7\x98\x6f\xbf\x5e\x05\x76\x9d\x04\xd2\x0b\xc7\x31\x94\x14\x4a\xa0\x94\xd2\xa4\x0f\x20\x4b\xe3\xb5\x88\x56\x32\xd2\x78\x77\x5d\xe3\x77\x2f\x92\x7f\x77\x37\x81\x14\x4a\xae\xea\x2b\x59\x67\x34\x73\xce\x9c\x93\x38\xaa\x15\xa6\x0b\x00\x80\xe8\x02\x9e\x8c\x51\x24\x4b\xe0\x46\x13\x93\x1a\x2d\x5c\x44\x01\x5b\x13\xf5\x50\x13\xfe\xfd\x57\x1a\x27\x49\x1a\x1d\x83\x45\xc5\x48\xee\xf0\x76\xc4\x84\x74\xa5\x62\x75\x0c\x52\x2b\xa9\x71\x95\x29\xc3\x9f\x27\x38\x33\x56\xa0\x5d\x65\x86\xc8\x6c\x63\xb8\x2a\x0f\x20\x0c\x11\x0a\xc8\x14\xe3\xcf\xb7\x9e\xcb\x43\x0e\xb5\xa9\x60\xcf\x34\x79\xd0\x41\xa5\x05\x5a\xa0\x02\xa1\x30\x3b\xb4\x2c\x53\x08\x84\x07\x1a\x38\xb6\x8b\x53\x19\x73\x74\x52\x30\x9e\x02\x3c\xe9\xd9\x49\x27\x33\xa9\x24\xd5\x31\x14\x52\x08\xd4\x13\xe1\xbd\x14\x54\xc4\x70\x7d\x73\x59\x1e\x66\x32\x18\x7f\xde\x58\x53\x69\xb1\xe2\x46\x19\x1b\xf7\xf4\xc7\x82\xfe\xf6\xff\x3c\xcf\xa7\x4b\x3f\x76\xc5\x94\xdc\xe8\x18\x38\x6a\x42\x3b\x61\x25\x13\x42\xea\x4d\x0c\x37\xe5\x01\x2e\xcf\x16\x66\x99\x90\x95\x8b\xe1\xa3\x67\x31\x82\xd1\x05\x7c\xef\xbd\x08\xeb\xa1\xb9\xfc\x15\x38\x44\xc0\x03\xdb\x96\x0a\x1d\x64\xa8\xcc\xfe\xbf\x61\x29\xc7\x36\xb2\xcc\x19\x55\xd1\xcc\xc6\x5f\x2b\xa9\x05\x1e\x62\xb8\xba\x3d\x59\xf1\x63\x61\xf6\xe7\xc3\xf6\x05\xea\xe0\xda\xd6\x54\x0e\xc1\xdb\x74\x54\xf4\x7a\xae\xe2\xe0\xe9\x9b\xbc\x09\x67\x85\x03\xa3\x24\xea\x33\xbc\x48\x8a\xeb\xf4\x11\x15\x72\x1f\x24\x6e\xbc\x60\x2f\xcc\x25\x51\x71\x9d\x2e\x12\x0a\x89\xe1\x8a\x39\x77\xb7\x0c\x3f\xcb\x2e\xf8\x09\x15\xc8\xc4\x70\xb6\xe9\x38\xb5\x69\x2c\xd3\x1b\x84\x0f\x54\x97\x10\xdf\xc1\xfa\xde\x6c\xcb\xa7\xba\x44\xd7\xb6\x63\x51\xdf\x21\x4d\x18\x14\x16\xf3\xbb\x65\xd3\xf8\xfa\xf5\xcf\x1f\x5f\xdb\x76\x99\xf6\x7f\xdf\xd8\x16\xdb\x36\x89\x58\x9a\x44\x54\xa4\x67\xaf\x1f\xb9\xb1\x78\x0c\x35\x0d\x6a\xd1\x0f\x4a\xa2\x81\x97\xaf\x99\xb1\xcd\x8c\xa8\xbb\xf3\x48\xd6\x2b\x0f\x6c\x87\x5d\x78\xd6\x73\xc6\x47\x1a\xbb\x0b\x01\x4d\x23\xf3\xee\xed\xfa\xde\x58\x8b\x9c\xa0\x6d\xfb\x65\xf1\xee\x62\xd9\x33\xf2\x9a\xfa\xc2\x6d\xb9\x7e\x10\xa8\x49\xe6\x12\x6d\xdb\x42\x92\xd9\x28\x4d\x84\xdc\x8d\x7b\xee\xfd\x5c\x76\x0a\x21\x86\xe1\x71\xf8\x5f\x7f\x51\x26\x63\x2a\x9c\x4f\x76\x3a\x92\x7b\xa1\x9b\x4f\xc7\x32\x7d\xb1\xbc\x5b\xc5\x6c\xc4\x67\x24\x26\xd5\x6b\xcd\x23\x21\x77\xe7\x8d\xba\xeb\x20\x66\xec\x65\x2a\xcb\x83\x85\x24\xa6\x07\x93\x2f\x27\x6e\x75\xbe\x24\x51\xc8\x59\x1a\xa2\xf9\x49\xa9\x7f\xa9\xec\xd8\xbe\x57\x1a\xff\x5a\x10\xff\x30\x84\x6f\x0b\xe0\x7b\x85\xef\x77\x00\x00\x00\xff\xff\x1b\x73\x42\x85\xe5\x07\x00\x00")

func templatesCompletionsHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesCompletionsHtml,
		"templates/completions.html",
	)
}

func templatesCompletionsHtml() (*asset, error) {
	bytes, err := templatesCompletionsHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/completions.html", size: 2021, mode: os.FileMode(436), modTime: time.Unix(1561666921, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesExampleHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x56\x6d\x6f\xdb\xb6\x13\x7f\x9f\x4f\xc1\x3f\x1b\xfc\x2d\x03\x35\x85\x2d\x2b\x96\xa6\x92\x87\x2e\xcb\x80\x01\xd9\x56\xa4\xe9\x80\x61\x18\x06\x8a\x3c\xdb\x8c\x29\x52\x25\xcf\x96\x05\x43\xdf\x7d\xa0\x24\x5b\xf2\x43\x8a\x65\x18\xdf\x90\x3a\xfe\xee\x7e\xf7\xe4\xa3\x93\xff\xfd\xf0\xeb\xed\xe3\xef\x1f\xee\xc8\x02\x73\x3d\xbd\x48\xc2\x46\x34\x37\xf3\x94\x82\xa1\x41\x00\x5c\x4e\x2f\x08\x21\x24\xc9\x01\x39\x11\x0b\xee\x3c\x60\x4a\x3f\x3d\xfe\x38\xb9\xa6\xc3\x2b\xc3\x73\x48\xa9\xe3\xe5\x24\x5b\xcd\x66\xe0\x28\x11\xd6\x20\x18\x4c\xe9\x76\xcb\x1e\x78\xf9\x7d\x23\xae\xeb\x9d\x96\x17\x4e\x15\x48\xb0\x2a\x20\xa5\x08\x1b\x8c\x9f\xf8\x9a\xb7\x52\x4a\xbc\x13\x29\x5d\x20\x16\xfe\x26\x8e\x85\x95\xc0\x9e\x3e\xaf\xc0\x55\x4c\xd8\x3c\x6e\x8f\x93\xaf\xd9\x57\xec\x1b\x96\x2b\xc3\x9e\x3c\x9d\x26\x71\xab\xdb\x99\xd7\xca\x2c\x89\x03\x9d\x52\x8f\x95\x06\xbf\x00\x40\x3a\x64\x13\xde\x53\xb2\x70\x30\xeb\x79\x72\xbe\x11\xd2\xb0\xcc\x5a\xf4\xe8\x78\x11\x3e\x02\xdf\x5e\x10\x5f\xb1\x2b\xf6\x26\xa8\xf6\xb2\xc6\x81\x60\x6c\x9a\xc4\x81\xb4\xe3\x47\x85\x1a\xa6\x77\x1b\x9e\x17\x1a\xc8\x84\x6c\xb7\xec\x27\xb9\xa9\x6b\x12\x87\xe3\xad\x5d\x19\xac\xeb\x24\x6e\x61\x5d\x46\x82\xa3\xed\x39\x2c\x26\xac\x73\x20\x90\x6c\xf7\xa2\xb0\x32\x2e\x96\x73\x67\x57\x46\x4e\x84\xd5\xd6\xdd\x90\x57\x6f\xaf\x67\xd9\xdb\xeb\x77\x7b\x58\xdd\xda\x8b\x3b\x83\x49\xdc\xd6\x31\xc9\xac\xac\xa6\x17\x89\x54\x6b\x22\x34\xf7\x3e\xa5\xa1\x46\x5c\x19\x70\x93\x99\x5e\x29\xb9\xab\xcd\x00\xe1\x6c\x49\x7b\x9f\x0e\x75\xf5\x24\x97\x93\x37\x83\xeb\x06\x52\x38\x38\x94\x84\xd5\xc4\x2c\xa1\xae\x0f\xb1\xf1\x01\x38\x89\xa5\x5a\x7f\x91\xec\xdb\x63\xb2\x00\x51\x32\xa5\xbe\xe0\x02\x5c\xa8\xc1\x81\x89\x06\xb3\xb8\x3a\xf5\x27\xe1\x5d\xed\xb7\x5b\xf6\x41\xf3\xaa\xcd\xe8\xa7\x87\xfb\xba\xa6\x04\xb9\x9b\x87\x2e\xff\x2b\xd3\xdc\x2c\xe9\xa9\x76\x63\x21\x5b\x21\x5a\xd3\xb5\x54\xfb\x41\x77\xee\x66\x68\x48\x86\x66\x52\x38\x95\x73\x57\xd1\x69\xcf\x91\xc4\x2d\xf6\x8c\x4f\x31\x3f\x97\x38\x35\x23\xf0\x99\xb0\xdf\x14\x94\x3f\x5b\x09\x84\x72\x8f\xf4\x28\x91\x27\x61\xc5\xd0\x76\xde\x77\x4a\x6e\xd2\x5d\xef\xfd\x7f\xad\xa0\x4c\x85\x0d\x17\xa8\xac\xf1\xcf\x84\xf6\xf2\xf0\x6e\x7b\x9b\xcf\xc7\xf7\x85\x18\x41\xfb\xe3\xde\x78\x49\x48\x21\x23\xff\x55\x28\xef\x3f\x3e\xfe\xbb\x10\x8c\x3c\x13\xc1\x8b\xb8\x9b\x4e\x16\xb6\xa8\xba\x19\x3a\xc9\xd0\x84\xdc\x16\x15\x69\x05\xcf\x3b\x76\x76\xbc\x90\x6e\xfc\xdc\x04\x59\x77\x66\x1f\xab\x3c\xb3\xba\xae\xd9\x40\x76\xb7\x29\x40\x20\x1c\x07\x90\xc4\xc7\x3f\x9d\xed\xb6\xe9\xc3\x01\x6e\xf0\x83\xeb\x8e\xbb\x6d\x38\x8e\x2f\x23\x69\xc5\x2a\x07\x83\x63\xe6\x80\xcb\x2a\x9a\xad\x8c\x08\xfd\x12\x8d\xfb\xf9\x76\x19\xd1\x57\xc7\xe1\x8f\x99\xd0\x4a\x2c\xcf\xe2\x09\x21\x01\xfe\x08\x1b\x8c\x2e\xa3\x51\x78\x87\xfe\x38\x79\x87\xfe\x1c\x8d\x19\x47\x74\x11\xed\x1e\x24\x3a\x1e\xef\x47\x65\xdd\x1d\xc3\xde\x1c\x76\x6e\x32\x2e\xe5\xdd\x1a\x0c\xde\x2b\x8f\x60\xc0\x45\xa3\x25\x54\xd2\x96\x66\xf4\x9a\xec\x7d\x81\xa1\x33\xbe\x54\x28\x16\x24\x02\xb6\x84\x6a\x7c\x34\xb5\x05\xf7\x40\xe8\x7b\xe7\x6c\x79\x0f\x33\xa4\x37\x27\x15\x2c\x95\x91\xb6\x64\xda\x0a\x1e\x6c\x33\x07\x85\xe6\x02\xa2\x66\x42\x39\x58\xb7\xb3\xa9\x77\x7d\xbf\x32\x07\x7c\xf9\xee\x39\xb6\x07\x35\x5f\xbc\x94\xee\x17\xd8\xe0\x3f\xa5\xab\x0f\xf3\xb7\x4b\x4d\x5f\x98\xf0\xd8\x0e\xb3\x21\xac\xf1\x56\x03\xd3\x76\x1e\x35\xcd\xae\xcc\x9c\x04\xd0\x0d\x7d\xdd\xec\x03\xce\x35\x77\x04\x34\x49\xfb\xb2\x08\x07\x1c\xe1\x4e\x43\xf8\x8a\x46\x01\xcf\x1d\xf0\xd1\x40\x09\x34\x5b\x73\xbd\x02\x92\x36\xe6\xfa\x8b\xbd\x91\xf0\x0e\x32\x5e\x14\x60\xe4\xed\x42\x69\x19\x81\x3e\xd4\xf7\xa0\x41\x60\x34\x3e\xa3\x0b\x1b\x10\xb7\x36\xcf\xb9\x91\xad\xfb\xf4\x1c\xaa\x61\x70\x90\xdb\x35\x1c\x31\xd4\x17\xfd\x1f\x95\x24\xee\x5e\xe4\xb8\xfd\x03\xf6\x77\x00\x00\x00\xff\xff\xfd\x85\xed\x76\x91\x09\x00\x00")

func templatesExampleHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesExampleHtml,
		"templates/example.html",
	)
}

func templatesExampleHtml() (*asset, error) {
	bytes, err := templatesExampleHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/example.html", size: 2449, mode: os.FileMode(436), modTime: time.Unix(1561486938, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesExamplesHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x56\x5f\x6f\xdb\x36\x10\x7f\xcf\xa7\xb8\x72\x29\xd0\x3e\x44\x04\xd6\x0c\x5d\x3d\x59\xc3\x9a\xa6\x40\x81\x62\x0b\x92\xf4\x61\x8f\x94\x44\x5b\x44\x68\x52\x20\xcf\x89\x0d\x41\xdf\x7d\xa0\x48\xfd\xb1\x2a\xa9\x4b\xa3\x07\x5b\xbc\xfb\xf1\xfe\xfe\x8e\x54\xfc\xea\xd3\x3f\x57\xf7\xff\xde\x5c\x43\x81\x3b\x99\x9c\xc5\xee\x0f\x24\x53\xdb\x35\xe1\x8a\x38\x01\x67\x79\x72\x06\x00\x10\xef\x38\x32\xc8\x0a\x66\x2c\xc7\x35\xf9\x76\xff\xf9\xe2\x77\x12\x54\x52\xa8\x07\x30\x5c\xae\x89\xc5\xa3\xe4\xb6\xe0\x1c\x09\xe0\xb1\xe4\x6b\x82\xfc\x80\x34\xb3\x96\x40\x61\xf8\x66\x4d\x0a\xc4\xd2\xae\x28\xdd\xb1\x43\x96\xab\x28\xd5\x1a\x2d\x1a\x56\xba\x45\xa6\x77\xb4\x13\xd0\x77\xd1\xbb\xe8\x37\xb7\xb5\x97\x45\x3b\xa1\x22\x67\x2c\x89\xa9\x73\x1a\xfc\xa3\x40\xc9\x93\xeb\x03\xdb\x95\x92\x5b\xb8\x80\xaa\x8a\x6e\x18\x16\x75\x1d\x53\xaf\xf3\xb8\x26\x3a\xff\xee\x1e\x64\xa9\xe4\x50\x75\x6b\xf7\x3c\x89\x1c\x8b\x15\xb0\x3d\x6a\x78\x25\x76\xa5\x36\xc8\x14\xfe\xd1\x61\xea\xee\x2d\xca\xb4\x31\x3c\xc3\x91\x81\x94\x65\x0f\x5b\xa3\xf7\x2a\xbf\xc8\xb4\xd4\x66\x05\xbf\xbc\xbf\xe4\xf9\xfb\xcb\x49\x1b\x29\xb7\xff\xc3\x40\xf6\x61\x93\x65\x1f\xc6\x06\x62\x1a\xf2\x89\xa9\x6f\x53\x9c\xea\xfc\x98\x9c\xc5\xb9\x78\x84\x4c\x32\x6b\xd7\x24\xd3\x0a\x99\x50\xdc\xb4\xad\x1a\xe8\x8c\x7e\x22\x7d\x31\xe2\xe2\x32\xa9\xaa\xe8\x4a\xef\x15\xd6\x35\xe8\x3d\x82\xde\xb8\x42\xde\x6b\x64\xb2\x15\x77\x35\x16\x6a\x58\xe4\xe2\xb2\x37\x54\x55\x62\x03\xd1\xdd\x71\x97\x6a\x59\xd7\x27\xa9\x39\x1f\x77\x85\x7e\x12\x6a\x0b\x1b\x6d\xc0\x36\xa0\x95\xb3\xd4\xe2\xe1\x4d\xcc\x02\x53\x3a\xe9\xb7\xdb\xaf\x75\x4d\x12\x26\x65\x4c\x59\xf2\x76\x00\xa1\x24\x49\x8f\xc1\x4c\xa3\x1b\x87\xc2\xa5\xe5\x0b\x41\x30\x29\x81\xb7\x29\x3d\xc7\xac\xca\xeb\xfa\xac\xaf\x9d\x67\x52\xa8\xab\x5f\x34\xbf\x17\x16\x8d\x28\x79\x3e\xa8\xb3\xc7\xf7\x73\xd5\xcb\xcc\xa9\x20\x00\x93\x1b\xa3\x1f\x45\xce\x4d\x4c\xb1\x98\x46\x7c\xe4\x16\xe7\xb5\x57\x9e\xa6\xf3\x80\xa6\xc1\xdf\xab\x63\x3a\x8e\xc8\x61\x26\xe2\xf6\xac\x1b\xca\xaa\xca\x30\xb5\xe5\x70\x5e\xa6\xb0\x5a\x43\xf4\xf1\xd8\x26\x31\xea\xc5\x6c\xe2\x5e\x91\x27\x55\x75\x5e\xa6\x51\xbf\x3b\xa6\x98\xcf\xa2\x3d\xf5\xdc\x86\x2f\xd6\xd5\xa4\xae\x43\x47\xdc\x94\x91\xd0\xb5\xe9\xdd\x8d\x85\x9e\x78\xe7\x8d\xcb\x86\x77\xe3\x08\x48\x88\xc9\x3b\x80\x37\x55\x55\x1a\xa1\x70\x03\xe4\x75\xf4\xeb\x86\x40\xab\xbb\xe1\x26\xe3\x6e\x66\x5e\xbf\x75\x14\x9a\x8e\x79\x29\x9b\xe0\x27\xb4\x6f\xce\x55\x50\x9f\x78\xfb\xb1\xd1\xa6\xe5\x73\xd5\xfc\xbe\xf1\x2d\xe1\x47\xa8\xd3\xc6\xc7\xb4\xa1\x7c\xf2\x8c\xb1\x08\xab\x42\x3f\x76\x07\x54\xbf\xf7\x19\x23\x32\xdd\x39\xd2\xde\x08\xae\xfe\xf3\xf4\xbf\x0b\x63\x3e\xa7\xbf\x3e\x94\x3c\x43\x9e\x4f\x23\x7a\xaa\x07\x86\x34\x84\xff\x2a\x2c\x0a\xb5\xb5\x1d\x6f\xec\x04\xf1\x97\xa3\x77\x8d\x1a\x91\x6e\x30\x05\x73\x09\xcd\x74\xea\xc5\x83\x2c\x7d\x42\xa7\xc9\xb5\x2f\xcf\x1d\xea\xfe\x9c\x0d\x67\xef\x9f\x22\x3f\xac\xab\xaa\xf5\x12\x7d\xf9\x34\xe8\x1e\x8c\x14\x21\xf9\x05\x8e\x9f\x14\x74\x70\x85\x0c\x0c\xb5\xd7\x4d\x53\xd8\xb1\xf0\xc7\x1e\x06\x9b\x5a\x7a\x2c\x9d\x4d\x5d\x19\x0d\xb7\xae\x84\xdd\xe6\x5b\x6e\xf7\x12\xe7\xc8\x01\x27\xc7\x9a\xe1\xb6\x3f\x0c\xba\xeb\xbd\x59\x13\x7f\xd1\x41\x0b\xfb\x89\xc3\xaf\xf7\xf1\xd9\x7d\x7a\x2c\x44\xe4\xd1\x0d\xf4\x96\xa9\x87\x05\xe4\xe4\xf5\x3b\x7e\xfe\xa6\x7f\x2d\x19\x58\x0c\x65\xa9\xe2\xd3\x3b\x5f\x7a\xb6\xf9\x45\x2e\x1e\xdd\x57\x57\xf8\x0b\x5f\x5d\xb4\xf9\x86\xfe\x2f\x00\x00\xff\xff\x82\x71\x84\xc1\x53\x0b\x00\x00")

func templatesExamplesHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesExamplesHtml,
		"templates/examples.html",
	)
}

func templatesExamplesHtml() (*asset, error) {
	bytes, err := templatesExamplesHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/examples.html", size: 2899, mode: os.FileMode(436), modTime: time.Unix(1561486938, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesRootHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x56\x51\x6f\x9b\x30\x10\x7e\xef\xaf\xb8\x7a\xed\xd4\x3e\x04\x4b\x6b\xa6\xae\x99\xa1\x52\xbb\x4e\xda\xd3\x22\xad\x7b\xd8\xa3\x01\xa7\xa0\x02\x46\xf6\x25\x6d\x84\xf8\xef\x93\xb1\x81\x34\x81\xac\x59\xa7\xe5\x21\x60\xdf\xf9\xbb\xbb\xef\x3e\x9f\x60\xc7\x5f\xbe\xdf\xde\xff\x9a\xdf\x41\x82\x79\x16\x1c\x31\xf3\x80\x8c\x17\x0f\x3e\x11\x05\x31\x1b\x82\xc7\xc1\x11\x00\x00\xcb\x05\x72\x88\x12\xae\xb4\x40\x9f\xfc\xbc\xff\x3a\xf9\x44\x9c\x29\x4b\x8b\x47\x50\x22\xf3\x89\xc6\x75\x26\x74\x22\x04\x12\xc0\x75\x29\x7c\x82\xe2\x19\x69\xa4\x35\x81\x44\x89\x85\x4f\x12\xc4\x52\xcf\x28\xcd\xf9\x73\x14\x17\x5e\x28\x25\x6a\x54\xbc\x34\x8b\x48\xe6\xb4\xdb\xa0\x17\xde\x85\xf7\xd1\x1c\xed\xf7\xbc\x3c\x2d\x3c\x03\x16\x30\x6a\x82\xba\xf8\x98\x62\x26\x82\xbb\x67\x9e\x97\x99\xd0\x30\x81\xaa\xf2\xe6\x1c\x93\xba\x66\xd4\xda\xac\x5f\x93\x9d\x7d\x37\x3f\xe4\x61\x26\xa0\xea\xd6\xe6\xf7\x94\xc6\x98\xcc\x80\x2f\x51\xc2\x71\x9a\x97\x52\x21\x2f\xf0\x73\xe7\x53\x77\x6f\x5e\x24\x95\x12\x11\x6e\x01\x84\x3c\x7a\x7c\x50\x72\x59\xc4\x93\x48\x66\x52\xcd\xe0\xdd\xe5\x54\xc4\x97\xd3\x41\x8c\x50\xe8\x57\x00\x44\x57\x8b\x28\xba\xda\x06\x60\xd4\xd5\xc3\xa8\x6d\x13\x0b\x65\xbc\x0e\x8e\x58\x9c\xae\x20\xca\xb8\xd6\x3e\x89\x64\x81\x3c\x2d\x84\x6a\x5b\xb5\x61\x53\xf2\x89\xf4\x64\xb0\x64\x1a\x54\x95\x77\x2b\x97\x05\xd6\x35\x74\x64\xa6\xc5\x26\x9b\xc9\xf4\xe5\x89\x33\xc6\x5d\x5f\xa9\x70\x27\x48\xa0\x13\xf9\x04\x3c\xcb\x18\xe5\xc1\xb9\x3d\xd3\x1f\xb2\xa4\xbb\x14\xec\xa2\xf9\x9f\x68\x54\x69\x29\xe2\x8d\x94\xac\x7f\x2f\xc1\x7e\x4f\xbd\xdc\x70\x8e\xc1\x5c\xc9\x55\x1a\x0b\xc5\x28\x26\xc3\x1e\x37\x42\xe3\xb8\xf5\xd6\x76\x74\xdc\xe1\x5e\x22\xcf\x76\xcd\x8c\x6e\x67\x64\x7c\x06\xf2\xb6\x0d\xda\xdc\xab\x2a\xc5\x8b\x07\x01\x27\x65\x08\x33\x1f\xbc\x9b\x75\x5b\x44\x5d\x0f\xa4\x30\x50\xb8\x35\xc4\x41\x55\x9d\x94\xa1\xd7\x9f\x66\x14\xe3\x51\x6f\xa8\xaa\x74\x61\x82\x7a\xdf\xb4\xe1\xa4\xae\x5d\x47\x8c\x20\x49\x55\x89\x22\xae\x6b\x07\x69\xed\x70\x56\x55\xa5\x4a\x0b\x5c\x00\x39\xf5\x3e\x2c\x08\xb4\xb6\xb9\x50\x91\x30\xa2\x39\x3d\xdf\x1b\xd3\xc1\x39\x92\xc7\x10\x9d\xf9\x40\xd0\xa6\x31\x63\x35\xef\xb6\xc7\x15\xb8\xdd\xb2\x97\xed\x61\xb4\x11\xe6\xff\x12\xef\x8f\x75\x1e\xca\x01\x71\x41\x27\xce\x65\x31\x22\xcd\x5e\x44\x4a\xae\x1a\x19\xb5\x32\xd0\x03\x2a\x6a\x01\xfb\xab\x7b\xad\xa5\x42\xdf\x30\xa9\xe4\xaa\xae\x49\xd0\xbd\x42\xd8\x5c\x18\x1e\x8c\x05\xde\xe5\xf1\xef\x13\xea\xa3\x46\xfb\x2e\xe2\x48\xf3\xde\x7c\x03\x75\x7b\x03\x6d\x27\x0e\xbd\x7f\xbb\x93\xf0\x5a\x37\x40\x86\x58\x1d\x7a\x2d\x6a\xc3\xee\xc6\xda\xb1\xbb\x5f\xe2\x3a\x6c\x27\xf3\xb8\xeb\xd6\x28\x31\x67\xfe\xc4\x3a\x1c\x32\x0d\x46\x11\x1a\x94\x57\x56\xff\xbe\x93\xda\xe6\xac\x22\xfb\xc1\x6d\x75\x07\xce\xa2\xfd\xe9\x52\x3e\x1e\x72\x1f\xc5\xc3\x7a\x87\x37\xd1\xff\xef\x07\xe3\x58\x9e\x6f\x9d\x84\x76\x11\xa7\x2b\xf3\xd1\xe1\x1e\xee\xa3\x83\x36\x9f\x90\xbf\x03\x00\x00\xff\xff\xb4\x01\x2b\x3c\x52\x0a\x00\x00")

func templatesRootHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesRootHtml,
		"templates/root.html",
	)
}

func templatesRootHtml() (*asset, error) {
	bytes, err := templatesRootHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/root.html", size: 2642, mode: os.FileMode(436), modTime: time.Unix(1561486938, 0)}
	a := &asset{bytes: bytes, info: info}
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
	if err != nil {
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
	"templates/ast.html": templatesAstHtml,
	"templates/completions.html": templatesCompletionsHtml,
	"templates/example.html": templatesExampleHtml,
	"templates/examples.html": templatesExamplesHtml,
	"templates/root.html": templatesRootHtml,
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
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"templates": &bintree{nil, map[string]*bintree{
		"ast.html": &bintree{templatesAstHtml, map[string]*bintree{}},
		"completions.html": &bintree{templatesCompletionsHtml, map[string]*bintree{}},
		"example.html": &bintree{templatesExampleHtml, map[string]*bintree{}},
		"examples.html": &bintree{templatesExamplesHtml, map[string]*bintree{}},
		"root.html": &bintree{templatesRootHtml, map[string]*bintree{}},
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

