package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	var (
		pkg     string
		data    string
		offsets string
		output  string
	)

	flag.StringVar(&data, "data", "", "datadeps file to embed into bindata")
	flag.StringVar(&offsets, "offsets", "", "offsets file")
	flag.StringVar(&output, "output", "datadeps-bindata.go", "output file")
	flag.StringVar(&pkg, "pkg", "main", "package name")
	flag.Parse()

	dataBuf, err := ioutil.ReadFile(data)
	if err != nil {
		log.Fatalln(err)
	}

	offsetsBuf, err := ioutil.ReadFile(offsets)
	if err != nil {
		log.Fatalln(err)
	}

	t, err := template.New("datadeps").Parse(_template)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Create(output)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	tmplData := templateData{
		Pkg:     pkg,
		Data:    template.HTML(dataBuf),
		Offsets: template.HTML(offsetsBuf),
	}

	err = t.Execute(f, tmplData)
	if err != nil {
		log.Fatalln(err)
	}
}

type templateData struct {
	Pkg     string
	Data    template.HTML // NOTE: we use template.HTML here so the template engine doesn't escape anything (+'s were getting escaped).
	Offsets template.HTML
}

var _template = `package {{.Pkg}}

import (
	"reflect"
	"unsafe"
)

// Datadeps returns the raw data backing datadeps
func Datadeps() ([]byte, error) {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&_datadeps))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(_datadeps)
	bx.Cap = bx.Len
	return b, nil
}

// Offsets returns the offsets gob data data for datadeps
func Offsets() ([]byte, error) {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&_offsets))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(_offsets)
	bx.Cap = bx.Len
	return b, nil
}

var _datadeps = []byte("{{.Data}}")

var _offsets = []byte("{{.Offsets}}")
`
