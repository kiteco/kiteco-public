package pythondocs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kennygrant/sanitize"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"golang.org/x/net/html"
)

var (
	stdlibDocs = []string{
		"stdtypes.html",
		"functions.html",
		"constants.html",
	}
)

// DocstringParser contains logic to take docstrings in html form and construct LangEntity objects out of it.
type DocstringParser struct {
	StructuredParser *StructuredParser
}

// NewDocstringParser constructs a new DocstringParser.
func NewDocstringParser(graph *pythonimports.Graph) *DocstringParser {
	return &DocstringParser{
		StructuredParser: NewStructuredParser(NewHTMLNormalizer(graph)),
	}
}

// Parse takes a node ID and its corresponding docstring in html
// form and returns a LangEntity constructed with information from
// the node and structured docstring.
func (d *DocstringParser) Parse(node *pythonimports.Node, argSpec *pythonimports.ArgSpec, html string) (*LangEntity, error) {
	entity := &LangEntity{StructuredDoc: &StructuredDoc{}}
	if node == nil {
		return nil, errors.New("Cannot parse nil node")
	}
	entity.NodeID = node.ID
	if kind := nodeToKind(node.Classification); kind != UnknownKind {
		entity.Kind = kind
	}
	// find module name, identifier and selector
	if !node.CanonicalName.Empty() {
		parts := node.CanonicalName.Parts
		entity.Module = parts[0]
		switch len(parts) {
		case 1: // module
			entity.Ident = parts[0]
			entity.Kind = ModuleKind
		default: // everything else
			entity.Ident = strings.Join(parts[:len(parts)-1], ".")
			entity.Sel = parts[len(parts)-1]
		}
		entity.StructuredDoc.Ident = entity.FullIdent()
	} else {
		// put together name from parent's information
		parent, _ := node.Attr("__objclass__")
		if parent != nil {
			var canonical, member string
			canonical = strings.TrimSpace(parent.CanonicalName.String())
			for attr, child := range parent.Members {
				if child != nil && child.ID == node.ID {
					member = attr
				}
			}
			if entity.Kind == UnknownKind {
				entity.Kind = AttributeKind
			}

			if canonical == "" {
				entity.Ident = member
				entity.StructuredDoc.Ident = member
			} else {
				entity.Module = strings.Split(canonical, ".")[0]
				entity.Ident = canonical
				entity.Sel = member
				entity.StructuredDoc.Ident = fmt.Sprintf("%s.%s", canonical, member)
			}
		} else {
			return nil, nil // cannot decipher the identifier
		}
	}

	entity.StructuredDoc.DescriptionHTML = d.StructuredParser.ParseStructuredDescription(html, entity.FullIdent())
	entity.Doc = synopsis(entity)

	// populate parameters from node's ArgSpec
	if argSpec != nil {
		for _, arg := range argSpec.Args {
			param := &Parameter{
				Name: arg.Name,
			}
			if len(arg.DefaultType) > 0 {
				param.Type = KwParamType
			} else {
				param.Type = RequiredParamType
			}
			entity.StructuredDoc.Parameters = append(entity.StructuredDoc.Parameters, param)
		}
		if len(argSpec.Vararg) > 0 {
			entity.StructuredDoc.Parameters = append(entity.StructuredDoc.Parameters, &Parameter{
				Name: argSpec.Vararg,
				Type: VarParamType,
			})
		}
		if len(argSpec.Kwarg) > 0 {
			entity.StructuredDoc.Parameters = append(entity.StructuredDoc.Parameters, &Parameter{
				Name: argSpec.Kwarg,
				Type: VarKwParamType,
			})
		}
	}
	if len(entity.StructuredDoc.Parameters) > 0 {
		params := d.StructuredParser.parseStructuredParams(entity.StructuredDoc.DescriptionHTML)
		for _, param := range entity.StructuredDoc.Parameters {
			if desc, ok := params[param.Name]; ok {
				param.DescriptionHTML = desc
			}
		}
	}
	return entity, nil
}

// DocParser contains logic to take a html doc and construct LangEntity objects out of it.
type DocParser struct {
	StructuredParser *StructuredParser
}

// NewDocParser constructs a new DocParser.
func NewDocParser(graph *pythonimports.Graph) *DocParser {
	return &DocParser{
		StructuredParser: NewStructuredParser(NewHTMLNormalizer(graph)),
	}
}

// ParseSphinxHTML takes a sphinx html document and returns a Module object
// containing the language entities detected in the html document.
func (d *DocParser) ParseSphinxHTML(r io.Reader, name string, coverage bool) *Module {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	// The module name can be in several places. Check them in sequence.
	var moduleName string
	doc.Find("span.target").Each(func(i int, s *goquery.Selection) {
		if moduleName != "" {
			return
		}
		attr, exists := s.Attr("id")
		if !exists {
			return
		}
		if strings.HasPrefix(attr, "module-") {
			moduleName = strings.TrimPrefix(attr, "module-")
		}
	})
	if moduleName == "" {
		doc.Find("div.section").Each(func(i int, s *goquery.Selection) {
			if moduleName != "" {
				return
			}
			attr, exists := s.Attr("id")
			if !exists {
				return
			}
			if strings.HasPrefix(attr, "module-") {
				moduleName = strings.TrimPrefix(attr, "module-")
			}
		})
	}
	if di := strings.Index(moduleName, "."); di != -1 {
		moduleName = moduleName[:di]
	}
	// Check if builtin
	if moduleName == "" {
		f := filepath.Base(name)
		for _, doc := range stdlibDocs {
			if f == doc {
				moduleName = "builtins"
				break
			}
		}
	}
	if moduleName == "" {
		log.Printf("Could not find module name: %s\n", name)
		return nil
	}

	// The module version is usually in the title, for docs obtained from ReadTheDocs.
	title := doc.Find("title").Text()
	split := strings.Split(title, " ")
	re := regexp.MustCompile("([0-9]+[a-z]*\\.?)+")
	var version string
	for _, s := range split {
		if v := re.FindString(s); len(v) > 0 {
			version = v
		}
	}

	if version == "" {
		log.Println("Could not find module version")
	}

	module := &Module{
		Name: moduleName,
		Documentation: &LangEntity{
			Module: moduleName,
			Ident:  moduleName,
			Kind:   ModuleKind,
		},
		Version: version,
	}
	d.extractVariables(module, doc, coverage)
	d.extractExceptions(module, doc, coverage)
	d.extractFunctions(module, doc, coverage)
	d.extractClasses(module, doc, coverage)
	d.extractClassMethods(module, doc, coverage)
	d.extractClassAttributes(module, doc, coverage)

	if moduleName == "builtins" {
		d.updateBuiltinIdents(module)
	}
	d.updateVersion(module, version)
	d.updateStructuredDoc(module)
	d.updateSynopsis(module)

	if coverage {
		// To ensure distinct output files
		outputName := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		renderCoverage(doc, outputName)
		log.Println(fmt.Sprintf("Wrote colored parser coverage doc to %s-coverage.html", outputName))
	}

	return module
}

func (d *DocParser) extractVariables(module *Module, doc *goquery.Document, coverage bool) {
	doc.Find("dl.data").Each(func(j int, v *goquery.Selection) {
		var sig, doc string
		dt := v.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(v)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.Vars = append(module.Vars, &LangEntity{
			Module:        module.Name,
			Kind:          VariableKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			v.AddClass("kite_highlight")
			v.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:Variable:<i>%s</i></b></h1>", id))
		}
	})
}

func (d *DocParser) extractExceptions(module *Module, doc *goquery.Document, coverage bool) {
	extExc := func(j int, e *goquery.Selection) {
		var sig, doc string
		dt := e.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(e)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.Exceptions = append(module.Exceptions, &LangEntity{
			Module:        module.Name,
			Kind:          ExceptionKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			e.AddClass("kite_highlight")
			e.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:Exception:<i>%s</i></b></h1>", id))
		}
	}
	doc.Find("div.section#exceptions").Each(func(j int, s *goquery.Selection) {
		s.Find("dl.class").Each(extExc)
	})
	doc.Find("dl.exception").Each(extExc)
}

func (d *DocParser) extractFunctions(module *Module, doc *goquery.Document, coverage bool) {
	doc.Find("dl.function").Each(func(j int, f *goquery.Selection) {
		var sig, doc string
		dt := f.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(f)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.Funcs = append(module.Funcs, &LangEntity{
			Module:        module.Name,
			Kind:          FunctionKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			f.AddClass("kite_highlight")
			f.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:Function:<i>%s</i></b></h1>", id))
		}
	})
}

func (d *DocParser) extractClasses(module *Module, doc *goquery.Document, coverage bool) {
	doc.Find("dl.class").Each(func(j int, f *goquery.Selection) {
		var sig, doc string
		dt := f.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(f)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.Classes = append(module.Classes, &LangEntity{
			Module:        module.Name,
			Kind:          ClassKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			f.AddClass("kite_highlight")
			f.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:Class:<i>%s</i></b></h1>", id))
		}
	})
}

func (d *DocParser) extractClassMethods(module *Module, doc *goquery.Document, coverage bool) {
	doc.Find("dl.method").Each(func(j int, f *goquery.Selection) {
		var sig, doc string
		dt := f.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(f)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.ClassMethods = append(module.ClassMethods, &LangEntity{
			Module:        module.Name,
			Kind:          MethodKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			f.AddClass("kite_highlight")
			f.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:ClassMethod:<i>%s</i></b></h1>", id))
		}
	})
}

func (d *DocParser) extractClassAttributes(module *Module, doc *goquery.Document, coverage bool) {
	doc.Find("dl.attribute").Each(func(j int, f *goquery.Selection) {
		var sig, doc string
		dt := f.Find("dt")
		if dt == nil {
			return
		}
		s, err := dt.Html()
		if err == nil {
			sig = cleanHTML(s)
		}
		doc = d.extractDescription(f)
		id, ok := dt.Attr("id")
		if !ok || id == "" {
			return
		}
		ident, sel := splitSelector(id)
		module.ClassAttributes = append(module.ClassAttributes, &LangEntity{
			Module:        module.Name,
			Kind:          AttributeKind,
			Ident:         ident,
			Sel:           sel,
			Signature:     sig,
			SignatureHTML: s,
			DocHTML:       doc,
		})
		if coverage {
			f.AddClass("kite_highlight")
			f.PrependHtml(fmt.Sprintf("<h1 class='kite_header'><b>Kite:ClassAttr:<i>%s</i></b></h1>", id))
		}
	})
}

func (d *DocParser) extractDescription(f *goquery.Selection) string {
	if d, err := f.Find("dd").Html(); err == nil && d != "" {
		return d
	}

	// Introduce temporary parent around each sibling, so we get the
	// entire original sibling when calling .Html(), instead of just its descendents.
	// WrapHtml returns the original unwrapped set of siblings, so we ignore
	// it's return value.
	f.NextUntil("dl").WrapHtml("<div></div>")

	var html string
	f.NextUntil("dl").AddClass("kite_highlight").Each(func(j int, g *goquery.Selection) {
		h, err := g.Html()
		if err != nil {
			log.Printf("Error parsing description of %s: %v", g.Text(), err)
		}
		html += h
	})
	return html
}

func (d *DocParser) updateBuiltinIdents(m *Module) {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, c := range categories {
		for _, entity := range c {
			entity.Ident = builtinIdent(entity.Ident)
		}
	}
}

func (d *DocParser) updateVersion(m *Module, version string) {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, c := range categories {
		for _, entity := range c {
			entity.Version = version
		}
	}
}

func (d *DocParser) updateStructuredDoc(m *Module) {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, c := range categories {
		for _, entity := range c {
			entity.StructuredDoc = d.StructuredParser.parseStructuredDoc(entity)
			entity.DocHTML = "" // no longer need this field's value
		}
	}
}

func (d *DocParser) updateSynopsis(m *Module) {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, c := range categories {
		for _, entity := range c {
			entity.Doc = synopsis(entity)
		}
	}
}

// StructuredParser contains logic to take a LangEntity and extract structure out of it.
type StructuredParser struct {
	HTMLNormalizer *HTMLNormalizer
}

// NewStructuredParser constructs a new structured parser.
func NewStructuredParser(normalizer *HTMLNormalizer) *StructuredParser {
	return &StructuredParser{
		HTMLNormalizer: normalizer,
	}
}

// parseStructuredDoc takes a LangEntity and returns a StructuredDoc
// object containing parsed structural information.
func (s *StructuredParser) parseStructuredDoc(entity *LangEntity) *StructuredDoc {
	structured := &StructuredDoc{
		Ident: entity.FullIdent(),
	}
	if len(entity.DocHTML) > 0 {
		structured.DescriptionHTML = s.ParseStructuredDescription(entity.DocHTML, entity.FullIdent())
	}
	if len(entity.Signature) > 0 {
		structured.Parameters = s.parseSignature(entity.Signature)
	}
	return structured
}

// ParseStructuredDescription normalizes the given html to our custom schema.
func (s *StructuredParser) ParseStructuredDescription(description, ident string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(description))
	if err != nil {
		log.Printf("Error parsing description: %v\n%s", err, description)
		return ""
	}
	return s.HTMLNormalizer.parseNormalizedDescription(doc.Find("body"), ident, rootDepth)
}

func (s *StructuredParser) parseStructuredParams(description string) map[string]string {
	params := make(map[string]string)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(description))
	if err != nil {
		log.Printf("Error parsing description: %v\n%s", err, description)
		return params
	}
	s.HTMLNormalizer.parseParams(doc.Find("body"), params)
	return params
}

func (s *StructuredParser) parseParameter(param string) *Parameter {
	param = strings.TrimSpace(param)
	if strings.HasPrefix(param, "**") {
		return &Parameter{
			Type: VarKwParamType,
			Name: strings.TrimSpace(param[2:]),
		}
	}
	if strings.HasPrefix(param, "*") {
		return &Parameter{
			Type: VarParamType,
			Name: strings.TrimSpace(param[1:]),
		}
	}
	if split := strings.Split(param, "="); len(split) > 1 {
		return &Parameter{
			Type:    KwParamType,
			Name:    strings.TrimSpace(split[0]),
			Default: strings.TrimSpace(split[1]),
		}
	}
	return &Parameter{
		Type: RequiredParamType,
		Name: strings.TrimSpace(param),
	}
}

func (s *StructuredParser) parseSignature(signature string) []*Parameter {
	parameters := []*Parameter{}
	begin := strings.Index(signature, "(")
	end := strings.Index(signature, ")")
	if begin == -1 || end == -1 {
		return parameters
	}

	signature = signature[begin+1 : end]

	var optional string
	required := signature
	if sep := strings.Index(signature, "["); sep != -1 {
		required = signature[:sep]
		optional = signature[sep:]
	}

	if len(required) > 0 {
		split := strings.Split(required, ",")
		for _, name := range split {
			parameters = append(parameters, s.parseParameter(name))
		}
	}

	if len(optional) > 0 {
		optional = strings.Replace(optional, "[", "", -1)
		optional = strings.Replace(optional, "]", "", -1)
		split := strings.Split(optional, ",")
		for _, name := range split {
			if len(name) == 0 {
				continue
			}
			param := s.parseParameter(name)
			if param.Type == RequiredParamType {
				param.Type = OptionalParamType
			}
			parameters = append(parameters, param)
		}
	}
	return parameters
}

// --

// splitSelector returns the identifier and selector of a selector expression.
// The module name is assumed to be at the head of `name`.
func splitSelector(name string) (x, sel string) {
	parts := strings.Split(name, ".")
	switch len(parts) {
	case 1:
		sel = parts[0]
	default:
		x = strings.Join(parts[:len(parts)-1], ".")
		sel = parts[len(parts)-1]
	}
	return
}

// nodeToKind gets a LangEntityKind from an import graph kind
func nodeToKind(kind pythonimports.Kind) LangEntityKind {
	switch kind {
	case pythonimports.Function:
		return FunctionKind
	case pythonimports.Type:
		return ClassKind
	case pythonimports.Module:
		return ModuleKind
	case pythonimports.Object:
		return VariableKind
	default:
		return UnknownKind
	}
}

func synopsis(entity *LangEntity) string {
	if entity.StructuredDoc == nil {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entity.StructuredDoc.DescriptionHTML))
	if err != nil {
		log.Printf("Error parsing synopsis for entity: %v\n%s", err, entity.FullIdent())
		return ""
	}
	body := doc.Find("body")
	result := strings.TrimSpace(parseSynopsis(body, "p"))
	if result == "" {
		result = raw(entity.StructuredDoc.DescriptionHTML)
	}
	return result
}

func parseSynopsis(f *goquery.Selection, selector string) string {
	var stop, found bool
	var synopsis []string
	f.Contents().Each(func(j int, g *goquery.Selection) {
		if isWhitespace(g) {
			return
		}
		if !found && g.Is(selector) {
			found = true
		}
		if found {
			if !g.Is(selector) {
				stop = true
			}
			if !stop {
				synopsis = append(synopsis, g.Text())
			}
		}
	})
	return strings.Join(synopsis, "\n")
}

// --

func raw(str string) (result string) {
	z := html.NewTokenizer(bytes.NewBufferString(str))
	for {
		tt := z.Next()
		switch tt {
		case html.TextToken:
			result += string(z.Text()) + " " // add extra space for readability
		case html.ErrorToken:
			if z.Err() != io.EOF {
				log.Printf("Error parsing html: %s", str)
				result = str
				return
			}
			result = cleanChars(result)
			return
		}
	}
}

func builtinIdent(ident string) string {
	builtin := "builtins"
	if len(ident) > 0 {
		builtin += "." + ident
	}
	return builtin
}

func cleanHTML(html string) string {
	return cleanChars(sanitize.HTML(html))
}

func isWhitespace(f *goquery.Selection) bool {
	return strings.TrimSpace(f.Get(0).Data) == ""
}

func cleanChars(text string) string {
	return strings.Replace(text, "\u00B6", "", -1)
}
