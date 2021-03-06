// Code generated by go-bindata.
// sources:
// templates/bundle-setup.sh
// templates/deploy-bundle.sh
// templates/provision.sh
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

var _templatesBundleSetupSh = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x18\xfd\x6f\xdb\x36\xf6\x77\xfd\x15\xef\x52\x63\xd9\xae\xa5\x94\xb4\xb9\xac\xcb\xe6\x01\x59\xec\x5e\x82\x25\x71\xe0\xa4\xeb\x86\xdd\x60\x50\xe4\xb3\xc4\x5a\x22\x79\x24\x65\xc7\x4d\xf3\xbf\x1f\x48\x49\xb6\x9c\x38\xfd\x18\x2e\x40\x00\xea\x7d\xf3\x7d\xf1\x3d\x3f\xfb\x47\x52\x59\x93\xa4\x42\x26\x28\xe7\x90\x52\x9b\x47\x16\x1d\x10\x8c\xa2\x67\xc0\x94\x9c\xa3\x14\x28\x19\x82\x75\xd5\x74\x1a\xd9\x8a\x2b\xa0\xda\x91\x0c\x1d\x54\x9a\x53\x87\x9b\x30\x21\xad\xa3\x45\x01\xb9\x53\x1a\xc8\xd2\x4b\x69\x41\x85\x48\x1d\x4a\xab\xcc\xb4\x50\x8b\xe8\x19\xdc\x8c\x06\xa3\x23\x58\x20\x30\x2a\x21\xa7\x73\x84\x69\x41\x33\xa0\x16\x9c\x82\x45\x8e\x2e\x47\xe3\x8f\x28\x69\x5a\x20\x08\x07\xca\x80\x54\x2e\x1a\x9c\x9f\x8e\xae\x6f\xfa\xb9\x73\xda\x1e\x25\x89\x75\xca\xd0\x0c\xe3\x4c\xa9\xac\x40\xaa\x85\x8d\x99\x2a\x93\xb5\xae\x64\x53\xf3\x9b\xb3\xf3\xe1\xe5\xf1\xc5\xb0\xbf\x01\x26\x4c\x57\xa4\x10\xb2\xba\x25\xb7\xaf\x0f\x27\x87\x07\x64\x3f\xde\xff\x57\xbc\x17\x3b\x6a\xe2\xec\x43\x84\x2c\x57\xb0\x33\x50\x0b\x59\x28\xca\x85\xcc\xa0\xd7\x0a\xda\x89\x58\x65\x0a\xe8\xd5\x76\x25\x2b\x38\x10\xb5\x26\x6a\x04\x9c\xd5\xce\xf0\xfc\xcc\x7b\xc4\x50\x23\xd0\xbe\x00\x97\x0b\x0b\x25\x5d\x82\xa3\x33\x04\x0a\x8b\x5c\x14\xb8\x53\xfb\xd6\x51\x03\xe4\x04\x42\xa4\x0a\xc5\x68\x91\x00\xb9\xfd\x30\xed\xc8\x36\x25\x90\xee\x77\x60\x2b\x38\x53\x72\x2a\xb2\x28\xba\xbb\x13\x53\x88\x4f\xde\x0e\x8e\xef\xef\xb7\x87\x8b\x2c\x41\x69\x94\xef\xf9\x8c\xbc\x26\xef\xf9\x0c\x32\xe1\x40\x2f\x5d\xae\x24\xe1\x38\x6f\x8e\xaf\x3a\x67\x22\xab\x52\x2f\x57\x88\xfa\x2b\xad\x44\xc1\x09\x5a\x8b\xd2\x09\x5a\xc0\x7f\x22\x00\x68\x19\xb4\xd0\x2b\xf2\xf5\x99\xcc\x85\x71\x15\x2d\x7c\xfa\xd9\x85\xc8\x5a\xf0\x22\x47\x0c\x39\xe3\x7d\x5b\x2b\xf6\xa7\xa8\x9c\x71\x61\x20\x99\x53\x93\xcc\x84\xc3\x24\xd3\x95\x2b\x75\xc4\xf8\x16\x98\x8f\x0a\x19\x81\x4f\x94\xa3\x24\xe1\x38\xc7\x42\x69\x34\x31\x6f\xa2\x18\xcb\xb9\xe0\x82\x86\x74\x61\xaa\xd4\x95\xc3\x84\x55\x9c\x26\x06\xb5\xb2\x49\x95\x56\xd2\x55\xfb\x87\x7b\x07\x49\x9d\x12\x01\x49\x3c\x92\xac\x71\x93\x1f\xe2\xbd\x78\xff\xfb\x43\xb2\x3f\xa1\x25\x3f\x3c\x88\x39\xa6\x6b\x27\xcf\x70\x09\x94\xcf\x81\x90\x29\x3a\x96\xfb\x6f\xfb\x7f\x34\xe8\xfb\x29\x7d\x49\xa7\xaf\xf7\x62\x5d\x35\x4a\xb9\x9e\x65\x40\x04\xc4\x7f\xc3\xd8\xcf\x15\x35\x59\x42\x10\xfa\x03\xd9\x8b\xa2\x85\xc7\xb4\x35\xe8\xdd\x4e\x84\x74\x68\x24\x2d\x88\x53\xaa\xb0\xb1\x7d\x45\x2a\x4b\x16\x68\x1d\xd9\x8f\x69\x49\x3f\x28\x49\x17\x75\x6d\xb2\x8a\x4b\x49\x7e\x88\xf7\xda\x8a\x3b\x3c\x20\xf3\xef\xe3\x57\xf1\x7e\xfc\x72\x2f\x76\xd9\x87\x4e\xe2\xdf\x7e\x98\x4f\xe1\x0b\x19\x98\x0e\x06\x26\x42\xb2\xa2\xe2\x58\xeb\x89\xf3\x6e\xe9\x74\xf1\x9b\x4c\x85\x48\x0f\x0f\x92\x90\x71\x5c\xca\x7f\x3e\x62\x0a\xf8\x86\x25\x2f\x15\x07\xfa\xdc\x3c\x25\xf9\x49\xcd\x0f\x94\x44\x75\x5b\xd8\xc5\x5b\xad\x8c\x83\xf3\xc1\xe4\xfc\xec\x97\xf1\xf1\xf8\x8f\xc9\xd5\xf1\xcd\x69\x7f\xa7\xf7\x00\x72\xb4\x55\xde\x23\x28\xde\x3a\x43\x6d\x72\xf2\xf6\xea\xe6\xac\x26\xd9\xd9\x85\x9f\x7f\xee\x94\x48\x5a\x49\x5e\xa0\x6f\xfa\xb1\xcd\x37\x8d\xf0\x7d\x62\x72\x3a\xba\x18\xf6\x1f\x88\xfd\x72\x11\x8d\xf1\x5b\x2d\x4e\x85\xfc\xb4\x31\x77\x77\x28\xf9\xfd\x7d\xdb\xb6\x06\x8a\xcd\xd0\x5c\x9c\xdf\xdf\x47\xcf\xe0\xac\x7d\x5e\x90\xf2\x02\xad\x05\x6e\xc4\x1c\xcd\x0b\xff\x8c\x70\x25\x77\x1d\x48\x44\x0e\xbf\x2b\x93\xbd\x00\x74\x2c\x8e\xe3\x26\x95\x39\x27\x3e\x9d\x43\x21\x09\xa7\xcc\xd2\x67\xb3\xd6\xf4\x28\x33\x54\xe7\x82\x59\x52\x8b\xb2\x89\xd6\x74\x95\xfe\x0f\xcb\xa1\x5b\x0a\x75\xad\x92\xd6\x14\x72\x70\xb0\xd7\xc2\x2a\x27\x8a\x00\xf0\xef\xdf\x1b\x65\x18\x36\x86\x82\xaf\x72\x58\x08\x97\xab\xca\x81\xc1\x54\x29\x57\x0b\x6f\x38\x6d\x29\xa2\xce\x3d\x79\xb8\x3c\xcc\x05\x0d\x9a\x4d\xc5\x9c\x50\xd2\xc2\xd4\xa8\xb2\x41\xee\x5a\x58\x60\x6a\xc5\x27\x8a\xd6\x83\x9c\xa1\xd2\xfa\xe0\x90\x50\xb3\xc0\x28\x61\x68\x9c\x98\x0a\x46\x1d\xda\xd0\x5c\x21\x93\x95\xce\x08\xcd\x50\x3a\xb0\x6a\xea\x16\xd4\x20\xd1\xc6\xf7\x28\x27\xd0\x12\xa6\xca\x52\xc9\xa6\xb1\x4e\xed\xf5\xf9\xaa\x01\xac\x3a\x58\x6d\x54\xa8\xf2\x50\xaa\x4d\xd3\x4a\x32\x9d\xc1\x47\x78\xd0\x17\x39\x90\x4f\x85\x27\x3c\x1f\x3b\x1c\x53\xf8\x93\x1a\x96\xf7\x43\xcb\xfa\xeb\x8b\x75\xd6\xfc\xbd\x6f\x0b\x9b\x4e\x0c\x16\x48\x2d\x02\x61\xf6\xbb\x1a\x6e\x9d\x1f\x2b\x76\xbe\xaa\xfb\xd5\x8a\x88\x0f\x67\x7b\x22\xac\x10\x7e\x50\x72\x54\x48\x34\x3c\x16\xaa\x1b\xbf\x26\xaa\x2b\x7c\x68\x8d\x33\xe1\x22\x2e\xac\x33\x22\xad\x7c\x38\xfb\xbd\x6f\x63\x48\xd0\xb1\x44\x59\xd2\x18\xfa\x63\xa8\xa7\xde\xd9\xa0\xf7\xdb\x70\x7c\x7d\x36\xba\x9c\x9c\x0d\xbe\x6b\x1c\x6f\x81\xac\x1d\xdf\xbc\x18\x99\x70\x79\x95\xc6\x42\x35\x00\x52\x1b\xe8\xdd\xee\x5d\xbd\xd5\xf3\x5f\x2f\xad\xd7\x35\x7b\x13\x17\x17\xc2\xba\x56\x8f\x43\xac\x2f\x44\xb5\x4b\xac\xaa\x0c\x43\x1b\x08\x62\xbe\x85\xeb\xab\x42\xf0\xa4\x47\x03\x83\x5d\x5a\x87\x25\x73\x05\x18\xb4\x8e\x1a\xd7\x04\xaa\x1b\x13\xdf\x72\x26\x65\x01\x42\x3a\x05\x2e\xc7\x75\xf4\xbc\x47\x0c\x4a\xd7\xef\x5d\xbd\x1b\x6c\x8e\x14\x75\x8b\x8a\x22\x46\x1d\xfc\xf4\x13\xec\x0e\x47\x6f\x42\x13\xab\xbb\xd3\x54\x14\x18\xbd\x19\x8f\x2e\xa0\x33\x75\xae\x8f\x47\x61\x94\x7c\x49\x32\x5d\x11\xbd\x7c\x15\x8d\xdf\x5e\x82\x9f\x80\x56\xd7\x22\x95\xce\x0c\xe5\xe8\xa1\x8f\xb0\x96\x09\xbd\x8c\xa2\x68\x78\xf9\x1b\x8c\xae\x86\x97\x17\x57\x67\x93\x26\x29\xfa\x07\xf1\x5e\xfc\x2a\x60\x2e\x4f\x4e\xce\x57\xe0\x97\xf1\x41\xfc\x9a\xec\x3f\xf7\x0d\x77\x7f\x2f\xde\x8b\x82\xd0\x4d\x0f\xc3\x37\xdf\x6c\xf3\x2f\x21\xb4\xf0\xe3\xb0\x2f\xae\x60\x93\x5d\x81\x58\x4e\x65\x86\x24\xc7\x82\x13\x4d\xd9\x8c\x66\x01\x29\x15\x69\xf8\x89\x41\xdf\x21\x50\x72\xdb\x8c\x7e\xfe\x6f\xfb\x50\xe8\xff\x58\xe9\x87\xdd\xf5\x77\xf6\xfc\x39\x39\x88\x5f\x77\x21\xc2\x75\xe9\x7d\xc2\xae\x3f\xc3\x04\xd2\xc1\x3e\xe8\x68\x6b\x4c\x21\x52\xc9\x58\xf1\xb2\xdf\xbb\xeb\xba\xe9\xfe\x31\x89\x9f\x33\x3f\x4d\xf5\x5e\x63\x16\xa6\xd1\x0d\xa8\x96\x5b\x80\x86\x97\x94\x95\xfb\x9b\x40\x91\xce\xd1\xa4\xb6\x0b\x6d\x40\xbe\xd1\xce\x05\x47\x63\xbb\xd9\x3a\xd2\x28\xe1\xe2\xea\x2c\x44\xb0\x19\x7e\x5d\xa9\x13\x3f\xae\x97\x5a\xf8\x28\xd6\x92\x7c\xba\x6e\x45\x6c\x0c\x6a\x8b\xc5\x22\xf6\x14\xa4\xd4\x22\x56\x26\x4b\xda\x36\x9f\xa8\x52\x8b\x64\x7e\x10\xef\xad\xfa\xaa\x6d\x65\x91\xde\xdd\x83\xc4\xbb\x6f\xd6\xa2\xb5\x16\x3f\xab\x7d\xb8\x9d\xc2\x57\xb0\x30\xfe\x09\xea\x35\x59\x9c\xd4\x7b\x4c\x65\x10\x08\xa9\xb7\x41\xa2\x8c\x43\x53\x49\xa2\x0d\x4e\xc5\x2d\x49\x97\x84\xe3\x94\x56\x85\x5b\xf3\x85\xe4\x22\xef\xa1\xf7\xad\xd4\x46\xb1\xef\xc0\xbb\x73\x13\xdb\xe6\xfd\x0a\xda\xee\x4c\x6b\x88\x5f\xad\xcc\x74\xc3\xb5\xdd\xf0\x9c\x2a\xa3\xe6\x8a\xbf\x00\x87\xa5\x56\x7e\x9b\x2b\x96\x50\x59\xbf\xdf\xf9\x39\xca\xaf\xcd\xa9\x0d\xb1\x5b\x89\xde\x3e\x12\x06\xc2\xb5\xda\xd3\xd1\x78\xf4\xdb\x68\x30\x79\x72\x18\x7b\x40\xf7\xef\xab\xb7\x93\xe3\xf3\xf3\xf1\x70\xf0\xf6\x64\xd8\xf7\xf9\xbb\x81\xfa\x65\x3c\x3a\x1e\x9c\x1c\x5f\xdf\xd4\xa8\x4d\xde\x77\x67\x37\xa7\x93\x9b\xe1\xe5\xf5\x68\xfc\xe6\x7c\xf4\xae\xbf\xbf\x81\x19\xbd\xbd\x99\x5c\xfd\x71\x33\x1a\x9f\x9c\x6e\xc1\x5c\xfc\x7e\x39\xbc\xe9\xb7\xd9\xbc\xd9\xd4\xa4\x22\x8c\xb2\x1c\x89\x4f\xda\xbc\x76\xd4\x63\x57\x3f\x4c\xf6\xeb\xeb\x53\x98\x2a\xe3\x73\xde\x6f\xff\xbe\xa1\x54\x32\x94\x34\xa4\xe8\x16\x88\x72\xdd\xb0\xed\x46\x5f\xdb\xe8\x62\xdb\xbb\x92\x8f\xa0\xb5\xb9\x7f\xb4\xfd\x8c\xd3\x7e\x5a\x34\x7e\x38\x5b\x27\x47\xa8\x33\xa2\xeb\x07\xc0\x54\x32\xb1\x36\xe7\xde\xd2\x63\xdf\x09\x57\x76\xfa\x07\x84\x16\xb3\xda\xce\xd6\xa6\xd5\x78\x47\xed\xcc\xe7\x81\xbf\x4c\xb8\xaa\x29\xa9\x7f\x37\x83\xcd\xfe\x1d\x09\x2f\xa4\xb5\xb9\xff\x9f\x34\xc9\xf1\x11\x32\x83\x1a\xc8\x1c\xae\x9d\x11\xcc\x9d\x2a\xeb\x7e\xc5\xe5\x49\x8e\x2c\x08\xfb\x79\x1b\x57\x2c\x71\xb1\x36\xbe\xfe\xa1\xc1\x9f\xb6\x8b\x90\x6a\x27\x8c\xdf\x9f\x93\x53\xce\x9f\xa4\xd9\x02\x8f\xa2\x77\xa3\xf1\xaf\x83\xb3\x31\x24\xd1\xf1\x60\x10\x5e\x59\xa6\xea\xc5\xb0\x5e\xe9\x93\xf6\xe1\x6d\x0f\xb5\x23\xf8\xea\xdb\xab\xde\xc8\x20\x84\xb8\x7e\xf6\xea\x7d\xe2\xd1\xc2\x90\x0a\x79\xd4\xbb\xf3\xc8\xfb\x9d\x40\xf7\x68\x7f\xea\x14\x4d\x21\xd2\xa3\xde\xdd\x03\x8a\xfb\x9d\xda\x88\x30\x09\x9d\xc3\xce\xdf\xfc\x4d\x69\xf3\x2b\xbc\xf3\x4f\xff\x94\xb4\x03\x1f\x1f\xff\xa4\xe3\xf7\xdc\x70\x85\x50\xf0\x27\xc7\x27\xa7\xc3\xfa\x0a\xa1\xd8\x43\x19\x45\xc3\xd1\x9b\xa8\x59\xf2\xeb\xb5\x40\x94\x34\xc3\xfa\x7d\x05\xe2\x3a\xb3\xc7\x51\x2d\x33\x7e\xb8\x4b\xfd\x2a\x1c\x36\x9b\x54\xeb\x64\x8e\x1a\x25\x47\xc9\x04\x5a\xa0\x92\x03\x33\x48\x1d\x26\x61\x51\xe9\xfc\x36\xe3\xb3\xb8\x8d\xdb\x53\x23\xbb\x5f\xa7\x38\x52\x6e\x25\x9d\x61\x58\xa4\x80\x2c\xbf\x7c\xba\x6b\x7e\x1c\x8a\x0f\xbb\x7a\xc9\x32\xda\x36\x89\x25\x4f\xe7\x57\xd4\xe5\xd6\x5d\xa9\x28\xe7\x51\x3d\x8d\x86\x73\xf8\xd9\x93\x32\x27\xe6\xde\xa4\x8d\xcc\x33\x60\xf0\xbf\x95\x30\x58\xa2\x74\x36\x76\xb7\x2e\x7a\x94\x99\x1c\x57\xbc\x2b\x37\x6f\x8e\x87\x8f\xad\x6e\x39\xfc\x9e\xdb\x58\xf2\xd4\x1a\x2c\xa6\xf0\x27\x90\xaf\xba\x79\xe2\x6f\x05\x7f\xfd\xe8\xa7\x5a\x19\x6a\xf8\x29\x1d\x9f\x91\xb1\xe9\x99\xa9\xa8\x53\xef\x73\x97\x0b\x03\xf7\xa7\x6e\xd6\xbd\x7d\x4a\x6d\xfe\x98\xc2\x54\xd2\x23\xb7\xaa\xcb\x55\x89\xed\x36\x19\x7b\xf6\x09\x2d\x04\xb5\x68\xbf\x48\x5f\xa0\x85\x1a\xd1\xdf\xdd\x92\x51\xbb\x0d\x89\x33\x54\xc8\xad\x14\xad\xd7\x42\x71\x11\x2d\x34\x16\x42\xa2\x4d\x0a\xbc\x15\xbe\xc3\x04\xce\xdd\x60\xfc\xff\x02\x00\x00\xff\xff\x32\x93\xac\x33\x58\x17\x00\x00")

func templatesBundleSetupShBytes() ([]byte, error) {
	return bindataRead(
		_templatesBundleSetupSh,
		"templates/bundle-setup.sh",
	)
}

func templatesBundleSetupSh() (*asset, error) {
	bytes, err := templatesBundleSetupShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/bundle-setup.sh", size: 5976, mode: os.FileMode(493), modTime: time.Unix(1610596329, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesDeployBundleSh = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8f\x4d\x6b\xc2\x40\x10\x86\xef\xf3\x2b\xa6\xea\x35\xd9\x7b\x41\xc1\x46\x4b\x05\x49\x0b\xea\x59\xd7\xcd\x68\x42\xc3\x6e\xd8\x99\x0d\xd6\x90\xff\x5e\xe2\x47\x15\x5a\x7b\xdc\x9d\xe7\x7d\x66\xde\xfe\x93\x0a\xec\xd5\xb6\xb0\x8a\x6c\x8d\x5b\xcd\x39\x00\x93\x60\x44\x00\x2f\xab\x74\x32\x9f\xae\x5f\x67\xf3\xe9\x50\xd5\xda\xab\xcf\x42\x48\x85\xaa\x74\x3a\x53\xdb\x60\xb3\x92\x62\xd1\x3e\xde\x1f\xaf\xe8\xc7\x78\xf9\x76\x43\x01\x44\x7b\x3c\x1c\xeb\x1d\x0e\xee\x5c\x18\x25\x3f\xef\x2e\x00\xd0\xad\xc5\xdb\x86\xb3\x5a\x31\x49\xa8\x62\xce\x81\x5d\xf0\x86\x7e\x03\x64\xeb\x6e\x0c\x64\x72\x87\x3d\x3a\x54\xce\x0b\xce\xd2\xc5\x72\x9c\x26\xd3\xf5\x6c\x32\xdc\x18\x2d\x77\xb1\xc2\xb2\x68\x6b\x28\x2a\xb2\x4d\x0f\x47\xa3\x87\xc6\xbf\x85\xc9\xfb\x2a\x5d\x3e\x74\x1a\x17\xac\xfc\xaf\x7d\xd4\x53\xb4\x97\xd3\xbc\x8f\x2c\xae\x42\xc9\x89\x09\x4d\x19\x58\xc8\x33\x3a\x6b\xa8\xfb\xc3\x33\x8f\xb9\x66\xe4\x60\x0c\x31\xef\x42\x59\x7e\xe1\xae\xb0\x05\xe7\x94\x41\xd3\x78\x6d\xf7\x84\x83\x4b\x16\x9f\x87\x18\x27\x25\x69\x1b\xaa\xe4\xa2\x6b\xdb\x4b\xbd\x85\xb8\xaa\x2a\xec\x1e\x9b\xe6\xca\xb7\x6d\x0f\xf4\x31\x78\x8a\xae\x82\xd3\x3d\xf7\x00\x34\x0d\xd9\xac\x6d\xe1\x3b\x00\x00\xff\xff\x2e\xb3\xd2\x82\x3a\x02\x00\x00")

func templatesDeployBundleShBytes() ([]byte, error) {
	return bindataRead(
		_templatesDeployBundleSh,
		"templates/deploy-bundle.sh",
	)
}

func templatesDeployBundleSh() (*asset, error) {
	bytes, err := templatesDeployBundleShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/deploy-bundle.sh", size: 570, mode: os.FileMode(493), modTime: time.Unix(1604974694, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesProvisionSh = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x54\x4d\x73\xdb\x36\x10\xbd\xf3\x57\x6c\xa3\x43\xda\x8e\x49\x4a\x8e\x3e\xec\x36\xce\xa5\x8d\xa7\x9d\x1c\x3c\xd3\xb4\xa7\x4c\x86\x03\x02\x4b\x11\x23\x10\xcb\xc1\x2e\x24\xab\x87\xfe\xf6\x0e\xc0\x8c\x2d\xab\x51\xdb\x8b\x69\xe1\x3d\xbc\xf7\x76\x17\xc0\xec\x9b\x3a\x72\xa8\x5b\xeb\x6b\xf4\x7b\x68\x15\xf7\x45\xc1\x28\x50\x62\x51\xcc\x20\xfd\x17\x47\xd8\x59\x41\x30\x36\x14\x1c\x0d\xc1\xb0\x33\x36\x40\x39\x42\xbd\x57\xa1\x4e\xd8\xa5\xf5\xda\xd1\xf6\x22\x16\x47\x47\xca\x4c\xb0\xee\xe9\xe0\xa1\xfc\x0d\x62\x1b\xbd\xc4\x6a\xfa\x9c\x18\x14\x5a\x09\xbc\x7d\x0b\xaf\xdf\x3f\xdc\xbf\x86\x77\x50\xa3\xe8\x9a\x8f\xac\xc5\x55\xa6\x5e\xcf\xcb\xc4\xaa\x34\xf9\xae\xe8\xb8\xea\xac\xc3\x72\x50\x8f\x70\x07\x8b\xf9\xf2\x66\xb5\x59\xa7\x55\x1f\x1a\x1a\xd1\x9f\x2c\x7a\x94\xca\x77\x8d\x26\xef\x25\x28\xbd\x6b\x5e\xee\x49\xb0\x1d\xf7\xcb\x4a\xf4\xd8\x0c\x38\xc0\x1d\x6c\x6e\xd6\xcb\x37\xd7\xb0\x58\xdf\x6e\x16\xab\x6b\x58\xdc\x2e\x57\x9b\xeb\x9b\x97\xcc\x30\x51\x97\xf3\xdb\xf5\xf4\x67\xb1\xde\x6c\x36\xd7\x8b\x33\xc1\xc3\x7f\xd3\xec\xd8\x38\xd2\xca\x35\x23\x05\x69\x82\xf2\x5b\xcc\xf1\xae\x97\xb0\x5e\xad\xde\xac\xa6\x02\x50\x3a\xeb\x04\xc3\xcb\x52\x92\x85\xd8\x01\x29\x4a\xfe\x36\x07\x65\x25\xef\xfe\xbf\xbb\x90\x45\xb5\xce\x72\x8f\x06\xee\x60\x3d\x9f\xbf\xcc\xcf\x8e\x0e\x0d\x8b\x0a\xd2\xa8\x4e\x30\x34\xd6\xb8\x14\xef\x8c\x26\x87\x26\xa0\x3e\xea\x8b\x58\xe4\x67\x44\x53\xc0\x8a\x69\x50\x8f\x29\x52\x72\xcd\x65\xbe\x7f\xb8\x2f\x0a\xd4\x3d\xc1\xab\x8f\x28\x62\xfd\x36\x9d\x4a\x63\x79\xc7\xaf\xce\xcf\x97\x51\xa2\x0a\xdb\xc1\xa7\x4f\x50\x22\xd4\x83\x17\xf8\xfc\xf9\x47\x90\x1e\x7d\x01\x00\x30\x03\xeb\x59\x94\xd7\x08\xbd\x62\xc8\xfd\x05\x16\x0a\x6a\x8b\x99\x90\xf5\xc2\x90\xf4\x26\xb1\xa7\xc5\xc9\x24\x49\x9e\x01\x97\x8e\xef\x3f\x99\xce\x43\xc9\xcf\xc0\x17\x07\x74\xcf\x81\x0d\xee\x6b\x36\xfa\x5f\x43\xab\x5c\x3b\x0c\x14\xbd\xa0\x39\x0d\xd8\x31\x94\x02\xf8\x28\xcb\x27\xa5\x13\x38\xf1\x9f\x1d\x26\xef\xce\x7e\xad\x83\x27\xd7\xfa\x52\x71\x99\x96\x1e\x89\x38\x1a\x25\x98\xc2\x02\x73\x0f\xe9\x12\xda\x2d\x08\x81\x8a\x42\x83\x12\xab\x95\x73\x47\x68\x11\x54\xeb\x30\x01\x89\x66\xbd\x10\x90\xf4\x18\x40\xbb\xc8\x82\xe1\xa9\x46\x86\x83\x95\x9e\xa2\xc0\x18\x68\x18\xd3\xbc\x8b\x19\x78\xca\x26\x4a\xb2\xd3\x0e\x8f\x60\x85\xd1\x75\xf0\xed\x5f\x75\xc5\xdc\xe7\xc8\xa5\xc1\x7d\xa9\xfe\x8c\x01\xbf\x03\xcb\x30\xbd\x31\x68\x60\x6f\x55\x72\xbd\x02\xe5\xc8\x6f\xb3\x7e\x96\x69\xa3\x37\x0e\xaf\x8a\x19\x28\x86\x51\x05\x01\xea\x32\x60\x70\x74\x74\x04\x8e\xad\xa6\x61\x50\xde\x14\x33\xf8\xfd\xe1\xe7\x87\x1f\x40\x7a\xcb\x49\x5b\x81\xb3\x22\x2e\x4d\x44\xef\x8e\x57\x80\x3c\x02\xdb\x34\xa2\xdc\x89\xd8\x7a\x94\xc4\xeb\x55\x30\x9a\x0c\x9a\xb3\x37\xec\x1d\xd4\x3d\x0d\x58\x4f\xfd\x9c\x4a\x98\x7a\x57\xfc\x42\x2c\xb0\x98\x57\xcb\x4d\xb5\xa8\xbe\xcf\x03\xfc\x83\x31\x7c\x99\x40\xfe\xfd\xab\x41\x2f\x56\x8e\xf7\xd6\x21\x7c\xb5\x01\x99\xf6\x51\x82\xd5\x92\xf4\x3e\xe0\xf1\xa7\x1e\xf5\x2e\x5d\x1e\x4f\x4f\x9a\x1f\x3c\x1d\x7c\xc2\x39\x2b\xe5\xc3\xe1\xa3\x73\xd3\x9d\x13\x8a\xba\x3f\x79\xb3\xc7\x40\x7b\xcb\x96\x3c\x9a\xe2\xef\x00\x00\x00\xff\xff\x8c\x61\x5a\xca\x3b\x06\x00\x00")

func templatesProvisionShBytes() ([]byte, error) {
	return bindataRead(
		_templatesProvisionSh,
		"templates/provision.sh",
	)
}

func templatesProvisionSh() (*asset, error) {
	bytes, err := templatesProvisionShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/provision.sh", size: 1595, mode: os.FileMode(420), modTime: time.Unix(1604974694, 0)}
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
	"templates/bundle-setup.sh": templatesBundleSetupSh,
	"templates/deploy-bundle.sh": templatesDeployBundleSh,
	"templates/provision.sh": templatesProvisionSh,
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
		"bundle-setup.sh": &bintree{templatesBundleSetupSh, map[string]*bintree{}},
		"deploy-bundle.sh": &bintree{templatesDeployBundleSh, map[string]*bintree{}},
		"provision.sh": &bintree{templatesProvisionSh, map[string]*bintree{}},
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

