package pythonlocal

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonindex"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var (
	// ErrUnknownNode indicates that a node was not created from this user's local index
	ErrUnknownNode = errors.New("node not found within this symbol index")
	// ErrUnknownPath indicates that a symbol's path was unknown to the local file manager
	ErrUnknownPath = errors.New("path for symbol not found")
	// ErrOffsetOutOfRange indicates that the position for a symbol was beyond the file length
	ErrOffsetOutOfRange = errors.New("offset out of range")
)

// Documentation represents a documentation struct extracted from local files.
type Documentation struct {
	CanonicalName string
	Path          string
	Identifier    string
	RawDocstring  string // RawDocstring is the raw doc string that we extract from the file
	Description   string // Description is the result of normalizing RawDocstring by removing indentation, quotation marks etc
	HTML          string // HTML is the html encoded version of RawDocstring
}

// Definition contains the python source that defines a class, function, or variable
type Definition struct {
	Path string
	Line int
}

// ArtifactMetadata holds information about an artifact and how it was produced.
type ArtifactMetadata struct {
	// The filename for which the artifact was requested.
	OriginatingFilename string
	// FileHashes for each file used to construct the artifact, filename => hash of file contents. Should not be mutated.
	FileHashes map[string]string
	// MissingHashes for each local file that could not be retrieved from storage when constructing the index, file content hash => true.
	MissingHashes map[string]bool
	// LibraryHashes for each library file used to construct the artifact, filename => hash of file contents. Should not be mutated.
	LibraryHashes map[string]string
	// MissingLibraryHashes for each local library file that could not be retrieved from storage when constructing the index, file content hash => true.
	MissingLibraryHashes map[string]bool
}

// A SymbolIndex is an index of python symbols
type SymbolIndex struct {
	// PythonPaths are import "roots" for e.g. libraries
	PythonPaths map[string]struct{}

	// SourceTree is the new representation of the local graph using pythontype.Value.
	SourceTree *pythonenv.SourceTree

	// DefinitionMap contains definitions keyed by canonical name.
	DefinitionMap map[string]Definition

	// DocumentationMap contains documentation keyed by canonical name.
	DocumentationMap map[string]Documentation

	// ArgSpecMap contains argspecs keyed by canonical name.
	ArgSpecMap map[string]pythonimports.ArgSpec

	// MethodMap contains method patterns
	MethodMap map[string]*pythoncode.MethodPatterns

	// ValuesCount contains counts of values used for ranking
	ValuesCount map[string]int

	// definitions is a diskmap containing the definitions fetched from S3
	Definitions *diskmap.Map
	Defcache    *lru.Cache

	// methods is a diskamp containing method patterns
	Methods      *diskmap.Map
	Methodscache *lru.Cache

	// InvertedIndex is used for active search
	// TODO(naman) unused: rm unless we decide to turn local code search back on
	InvertedIndex *pythonindex.Client

	// ArtifactRoot is root location of all diskmaps loaded in this SymbolIndex
	ArtifactRoot string

	// ArtifactMetadata contains information about the artifact for this index and how it was created.
	ArtifactMetadata ArtifactMetadata

	// LocalBuildTime stores a timestamp for when the artifact was built.
	LocalBuildTime time.Time
}

// LocateSymbol in the index
func (s *SymbolIndex) LocateSymbol(ctx kitectx.Context, id string) (pythontype.Value, pythontype.Value, string, error) {
	ctx.CheckAbort()
	if s.SourceTree != nil {
		ns, val, attr, err := s.SourceTree.LocateSymbol(ctx, id)
		if err == nil {
			return ns, val, attr, nil
		}
	}
	return nil, nil, "", fmt.Errorf("unable to locate symbol `%s`", id)
}

// FindSymbol in the index
func (s *SymbolIndex) FindSymbol(ctx kitectx.Context, file string, path []string, attr string) (pythontype.Value, pythontype.Value, error) {
	ctx.CheckAbort()
	if s.SourceTree != nil {
		ns, val, err := s.SourceTree.FindSymbol(ctx, file, path, attr)
		if err == nil {
			return ns, val, err
		}
	}
	return nil, nil, fmt.Errorf("unable to find symbol %s:%s::%s", file, strings.Join(path, "."), attr)
}

// Locate value in the index
func (s *SymbolIndex) Locate(ctx kitectx.Context, id string) (pythontype.Value, error) {
	ctx.CheckAbort()
	if s.SourceTree != nil {
		val, err := s.SourceTree.Locate(ctx, id)
		if err == nil {
			return val, nil
		}
	}
	return nil, fmt.Errorf("unable to locate value `%s`", id)
}

// FindValue ion the index.
func (s *SymbolIndex) FindValue(ctx kitectx.Context, file string, path []string) (pythontype.Value, error) {
	ctx.CheckAbort()
	if s.SourceTree != nil {
		val, err := s.SourceTree.FindValue(ctx, file, path)
		if err == nil {
			return val, nil
		}
	}
	return nil, fmt.Errorf("unable to find value %s:%s", file, strings.Join(path, "."))
}

// Package returns the parent package that contains the specified path.
// If the path is to a module this returns the package for the directory
// containing the module, if the path is to a directory this returns the package
// for the parent directory.
func (s *SymbolIndex) Package(ctx kitectx.Context, p string) (*pythontype.SourcePackage, error) {
	ctx.CheckAbort()
	if s.SourceTree != nil {
		pkg, err := s.SourceTree.Package(p)
		if err == nil {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("unable to find package for `%s`", p)
}

// Cleanup removes any temporary state associated with SymbolIndex (particularly diskmaps)
func (s *SymbolIndex) Cleanup() error {
	// Nothing to cleanup if ArtifactRoot is empty (for kite local)
	if s.ArtifactRoot == "" {
		return nil
	}
	err := os.RemoveAll(s.ArtifactRoot)
	if err != nil {
		log.Printf("error cleaning up artifact root %s: %s", s.ArtifactRoot, err)
	}
	return nil
}

// Documentation returns the doc string for the value
func (s *SymbolIndex) Documentation(ctx kitectx.Context, v pythontype.Value) (*Documentation, error) {
	ctx.CheckAbort()
	doc, found := s.DocumentationMap[LookupID(v)]
	if !found {
		return nil, ErrUnknownNode
	}

	return &doc, nil
}

// Definition returns a chunk of user code containing the definition for the given value
func (s *SymbolIndex) Definition(ctx kitectx.Context, v pythontype.Value) (*Definition, error) {
	ctx.CheckAbort()
	return s.definition(LookupID(v))
}

func (s *SymbolIndex) definition(key string) (*Definition, error) {
	start := time.Now()
	defer func() {
		definitionDuration.RecordDuration(time.Since(start))
	}()

	if s.DefinitionMap != nil {
		if def, found := s.DefinitionMap[key]; found {
			return &def, nil
		}
	}

	if s.Definitions != nil {
		if obj, ok := s.Defcache.Get(key); ok {
			definitionRatio.Hit()
			definitionCachedDuration.RecordDuration(time.Since(start))
			return obj.(*Definition), nil
		}
		definitionRatio.Miss()

		var def Definition
		err := diskmap.JSON.Get(s.Definitions, key, &def)
		if err == nil {
			s.Defcache.Add(key, &def)
			definitionErrRatio.Miss()
			return &def, nil
		} else if err == diskmap.ErrNotFound {
			definitionErrRatio.Miss()
		} else {
			log.Println("diskmap: get definition error:", err)
			definitionErrRatio.Hit()
		}
	}

	return nil, fmt.Errorf("definition not found")
}

// ArgSpec gets the argument spec for the given value
func (s *SymbolIndex) ArgSpec(ctx kitectx.Context, v pythontype.Value) (*pythonimports.ArgSpec, error) {
	ctx.CheckAbort()
	if s.ArgSpecMap == nil {
		return nil, ErrUnknownNode
	}
	key := LookupID(v)
	argspec, found := s.ArgSpecMap[key]
	if !found {
		return nil, ErrUnknownNode
	}
	return &argspec, nil
}

// ValueCount gets the count of a value in this index
func (s *SymbolIndex) ValueCount(ctx kitectx.Context, v pythontype.Value) (int, error) {
	ctx.CheckAbort()
	if s.ValuesCount == nil {
		return 0, ErrUnknownNode
	}
	count, found := s.ValuesCount[LookupID(v)]
	if !found {
		return 0, ErrUnknownNode
	}
	return count, nil
}

// MethodPatterns gets the method patterns for a value in this index
func (s *SymbolIndex) MethodPatterns(ctx kitectx.Context, v pythontype.Value) (*pythoncode.MethodPatterns, error) {
	ctx.CheckAbort()
	start := time.Now()
	defer func() {
		methodsDuration.RecordDuration(time.Since(start))
	}()

	if v == nil {
		return nil, fmt.Errorf("nil value")
	}

	key := LookupID(v)

	if s.MethodMap != nil {
		methods, found := s.MethodMap[key]
		if !found {
			return nil, fmt.Errorf("methodd patterns not found")
		}
		return methods, nil
	}

	if s.Methods != nil {
		if obj, exists := s.Methodscache.Get(key); exists {
			methodsRatio.Hit()
			methodsCachedDuration.RecordDuration(time.Since(start))
			mp, ok := obj.(*pythoncode.MethodPatterns)
			if !ok {
				return nil, fmt.Errorf("corrupted method pattern")
			}
			pythoncode.ProcessPatterns(mp)
			return mp, nil
		}
		methodsRatio.Miss()

		var mp pythoncode.MethodPatterns
		err := diskmap.JSON.Get(s.Methods, key, &mp)
		if err == nil {
			s.Methodscache.Add(key, &mp)
			pythoncode.ProcessPatterns(&mp)
			return &mp, nil
		} else if err == diskmap.ErrNotFound {
			methodsErrRatio.Miss()
		} else {
			log.Println("diskmap: get methods error:", err)
			methodsErrRatio.Hit()
		}
	}

	return nil, fmt.Errorf("method patterns not found")
}

// BuildDocumentation constructs a Documentation object from the body of a function, class, or module
func BuildDocumentation(filepath, identifier, canonical string, stmts []pythonast.Stmt) *Documentation {
	if len(stmts) == 0 {
		return nil
	}

	path, err := fromUnix(filepath)
	if err != nil {
		rollbar.Error(fmt.Errorf("error converting filepath for documentation"), err, filepath)
		return nil
	}

	// If the first statement is a string literal then it is the docstring
	var str string
	switch expr := stmts[0].(type) {
	case *pythonast.ExprStmt:
		if stringExpr, _ := expr.Value.(*pythonast.StringExpr); stringExpr != nil {
			str = stringExpr.Literal()
		}
	default:
		return nil
	}

	if str == "" {
		return nil
	}

	text := DedentDocstring(str)
	return &Documentation{
		Path:          path,
		Identifier:    identifier,
		CanonicalName: canonical,
		RawDocstring:  str,
		Description:   text,
		HTML:          "",
	}
}

// LookupID returns the string ID for finding a value in a map
func LookupID(v pythontype.Value) string {
	loc := pythonenv.Locator(v)
	if loc == "" {
		return ""
	}

	hash := fnv.New64()
	if _, err := hash.Write([]byte(loc)); err != nil {
		return ""
	}

	return fmt.Sprintf("%020d", hash.Sum64())
}

// BuildDefinition creates a definition from an AST node
func BuildDefinition(filepath string, node pythonast.Node, lines *linenumber.Map) *Definition {
	defBeginLine := lines.Line(int(node.Begin()))

	path, err := fromUnix(filepath)
	if err != nil {
		rollbar.Error(fmt.Errorf("error converting filepath for definition"), err, filepath)
		return nil
	}

	return &Definition{
		Path: path,
		Line: defBeginLine,
	}
}

// ArgSpecFromFunctionDef constructs an ArgSpec from a FunctionDefStmt
func ArgSpecFromFunctionDef(buf []byte, def *pythonast.FunctionDefStmt, id int64) *pythonimports.ArgSpec {
	argspec := pythonimports.ArgSpec{NodeID: id}

	for _, param := range def.Parameters {
		var name, defaultValue string
		name = "arg"
		if param != nil {
			if param.Name != nil {
				switch node := param.Name.(type) {
				case *pythonast.TupleExpr:
					// e.g. def foo((x,y,z)=(1,2,3)): ...
					name = "tuple"
				case *pythonast.NameExpr:
					name = node.Ident.Literal
				}
			}
			if param.Default != nil {
				defaultValue = string(buf[param.Default.Begin():param.Default.End()])
			}
		}
		argspec.Args = append(argspec.Args, pythonimports.Arg{
			Name:         name,
			DefaultValue: defaultValue,
			KeywordOnly:  param.KeywordOnly,
		})
	}

	if def.Vararg != nil {
		argspec.Vararg = def.Vararg.Name.Ident.Literal
	}
	if def.Kwarg != nil {
		argspec.Kwarg = def.Kwarg.Name.Ident.Literal
	}

	return &argspec
}
