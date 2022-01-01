package xmlvalidation

import (
	"fmt"
	"io/ioutil"
	"log"
	"unsafe"

	"github.com/jbussdieker/golibxml"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/krolaw/xsd"
)

// HTMLValidator allows validating a HTML file against the stored schema.
type HTMLValidator struct {
	Schema *xsd.Schema
}

// NewHTMLValidator constructs a new html validator given the schema file (.xsd).
func NewHTMLValidator(schemaPath string) *HTMLValidator {
	schema, err := fileutil.ReadFile(schemaPath)
	if err != nil {
		log.Fatalln(err)
	}
	xsdSchema, err := xsd.ParseSchema(schema)
	if err != nil {
		log.Fatalf("Error parsing XSD schema: %+v", err)
	}
	return &HTMLValidator{
		Schema: xsdSchema,
	}
}

// ValidateFile validates the given html file against the given html schema (.xsd) file.
// If it passes, error should be nil.
func (v *HTMLValidator) ValidateFile(inputFile string) error {
	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalln(err)
	}
	return v.Validate(string(input))
}

// Validate validates the given html against the given html schema (.xsd) file.
// If it passes, error should be nil.
func (v *HTMLValidator) Validate(html string) error {
	if html == "" {
		return fmt.Errorf("empty html")
	}
	doc := golibxml.ParseDoc(html)
	if doc == nil {
		return fmt.Errorf("error parsing XML doc")
	}
	defer doc.Free()

	// golibxml._Ctype_xmlDocPtr can't be cast to xsd.DocPtr, even though they are both
	// essentially _Ctype_xmlDocPtr.  Using unsafe gets around this.
	if err := v.Schema.Validate(xsd.DocPtr(unsafe.Pointer(doc.Ptr))); err != nil {
		return fmt.Errorf("error validating schema: %v", err)
	}
	return nil
}
