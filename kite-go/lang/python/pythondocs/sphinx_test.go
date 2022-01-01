package pythondocs

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errUnknownKind = errors.New("Unknown language entity kind")
	graph          *pythonimports.Graph
)

func TestMain(m *testing.M) {
	graph = pythonimports.MockGraph()
	os.Exit(m.Run())
}

func TestParseSphinx_Stdlib_String(t *testing.T) {
	filepath := "./test/stdlib_string.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()

	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "string"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: VariableKind, Module: mName, Ident: "string", Sel: "ascii_letters"},
		{Kind: VariableKind, Module: mName, Ident: "string", Sel: "ascii_lowercase"},
		{Kind: VariableKind, Module: mName, Ident: "string", Sel: "printable"},
		{Kind: ClassKind, Module: mName, Ident: "string", Sel: "Formatter"},
		{Kind: MethodKind, Module: mName, Ident: "string.Formatter", Sel: "format"},
		{Kind: ClassKind, Module: mName, Ident: "string", Sel: "Template"},
		{Kind: MethodKind, Module: mName, Ident: "string.Template", Sel: "substitute"},
		{Kind: AttributeKind, Module: mName, Ident: "string.Template", Sel: "template"},
		{Kind: FunctionKind, Module: mName, Ident: "string", Sel: "capwords"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_Celery(t *testing.T) {
	filepath := "./test/celery.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "celery"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: ClassKind, Module: mName, Ident: "celery", Sel: "Celery"},
		{Kind: ClassKind, Module: mName, Ident: "celery", Sel: "group"},
		{Kind: ClassKind, Module: mName, Ident: "celery", Sel: "chain"},
		{Kind: ClassKind, Module: mName, Ident: "celery", Sel: "chord"},
		{Kind: ClassKind, Module: mName, Ident: "celery", Sel: "signature"},
		{Kind: MethodKind, Module: mName, Ident: "celery.Celery", Sel: "close"},
		{Kind: MethodKind, Module: mName, Ident: "celery.Celery", Sel: "autodiscover_tasks"},
		{Kind: AttributeKind, Module: mName, Ident: "celery.Celery", Sel: "current_task"},
		{Kind: MethodKind, Module: mName, Ident: "celery.signature", Sel: "apply"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_Celery_Beat(t *testing.T) {
	filepath := "./test/celery_beat.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "celery"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: ExceptionKind, Module: mName, Ident: "celery.beat", Sel: "SchedulingError"},
		{Kind: ClassKind, Module: mName, Ident: "celery.beat", Sel: "ScheduleEntry"},
		{Kind: ClassKind, Module: mName, Ident: "celery.beat", Sel: "Service"},
		{Kind: MethodKind, Module: mName, Ident: "celery.beat.Service", Sel: "get_scheduler"},
		{Kind: MethodKind, Module: mName, Ident: "celery.beat.Service", Sel: "sync"},
		{Kind: AttributeKind, Module: mName, Ident: "celery.beat.Service", Sel: "scheduler"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_Validictory(t *testing.T) {
	filepath := "./test/validictory.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "validictory"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: FunctionKind, Module: mName, Ident: "validictory", Sel: "validate"},
		{Kind: ClassKind, Module: mName, Ident: "validictory", Sel: "SchemaValidator"},
		{Kind: ClassKind, Module: mName, Ident: "validictory", Sel: "ValidationError"},
		{Kind: ClassKind, Module: mName, Ident: "validictory", Sel: "SchemaError"},
		{Kind: ExceptionKind, Module: mName, Ident: "validictory", Sel: "ValidationError"},
		{Kind: ExceptionKind, Module: mName, Ident: "validictory", Sel: "SchemaError"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_Webassets(t *testing.T) {
	filepath := "./test/webassets.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "webassets"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: ClassKind, Module: mName, Ident: "webassets.filter.uglifyjs", Sel: "UglifyJS"},
		{Kind: ClassKind, Module: mName, Ident: "webassets.filter.less", Sel: "Less"},
		{Kind: ClassKind, Module: mName, Ident: "webassets.loaders", Sel: "YAMLLoader"},
		{Kind: MethodKind, Module: mName, Ident: "webassets.loaders.YAMLLoader", Sel: "load_bundles"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_Repoze(t *testing.T) {
	filepath := "./test/repoze.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "repoze"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: ClassKind, Module: mName, Ident: "repoze.who.plugins.auth_tkt", Sel: "AuthTktCookiePlugin"},
		{Kind: ClassKind, Module: mName, Ident: "repoze.who.plugins.basicauth", Sel: "BasicAuthPlugin"},
		{Kind: ClassKind, Module: mName, Ident: "repoze.who.plugins.sql", Sel: "SQLMetadataProviderPlugin"},
		{Kind: ClassKind, Module: mName, Ident: "repoze.who.middleware", Sel: "PluggableAuthenticationMiddleware"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_URLLib3(t *testing.T) {
	filepath := "./test/urllib3.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	parser := NewDocParser(graph)
	module := parser.ParseSphinxHTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}
	exp := "urllib3"
	act := module.Name
	if act != exp {
		t.Errorf("Module name %s did not match expected %s\n", act, exp)
	}
	mName := exp
	expLEs := []LangEntity{
		{Kind: ClassKind, Module: mName, Ident: "urllib3.poolmanager", Sel: "PoolManager"},
		{Kind: MethodKind, Module: mName, Ident: "urllib3.poolmanager.PoolManager", Sel: "clear"},
		{Kind: ClassKind, Module: mName, Ident: "urllib3.poolmanager", Sel: "ProxyManager"},
		{Kind: ExceptionKind, Module: mName, Ident: "urllib3.exceptions", Sel: "ClosedPoolError"},
		{Kind: ExceptionKind, Module: mName, Ident: "urllib3.exceptions", Sel: "HTTPWarning"},
	}
	for _, expLE := range expLEs {
		ok, err := searchModule(module, &expLE)
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Errorf("Could not find expected %+v in returned module\n", expLE)
		}
	}
}

func TestParseSphinx_StructuredDoc_Params(t *testing.T) {
	testCases := []string{
		"sys.exc_info()¶",                                          // no params
		"class HttpResponse[source]¶",                              // no params
		"sys.exit([arg])¶",                                         // one optional param
		"os.path.join(path1[, path2[, ...]])¶",                     // required and optionals (nested format)
		"render_to_response(name[, context][, instance])[source]¶", // required and optionals (chained format)
		"render(req, name[, context])[source]¶",                    // two required and one optional
		"boto.connect_s3(*args, **kwargs)¶",                        // variadic
		"class boto.S3Connection(aws_id=None)¶",                    // keyword
		"class datetime.delta([days[, weeks]])¶",                   // only optionals (nested format)
	}

	expected := [][]*Parameter{
		[]*Parameter{},
		[]*Parameter{},
		[]*Parameter{&Parameter{Type: OptionalParamType, Name: "arg"}},
		[]*Parameter{
			&Parameter{Type: RequiredParamType, Name: "path1"},
			&Parameter{Type: OptionalParamType, Name: "path2"},
			&Parameter{Type: OptionalParamType, Name: "..."},
		},
		[]*Parameter{
			&Parameter{Type: RequiredParamType, Name: "name"},
			&Parameter{Type: OptionalParamType, Name: "context"},
			&Parameter{Type: OptionalParamType, Name: "instance"},
		},
		[]*Parameter{
			&Parameter{Type: RequiredParamType, Name: "req"},
			&Parameter{Type: RequiredParamType, Name: "name"},
			&Parameter{Type: OptionalParamType, Name: "context"},
		},
		[]*Parameter{
			&Parameter{Type: VarParamType, Name: "args"},
			&Parameter{Type: VarKwParamType, Name: "kwargs"},
		},
		[]*Parameter{
			&Parameter{Type: KwParamType, Name: "aws_id", Default: "None"},
		},
		[]*Parameter{
			&Parameter{Type: OptionalParamType, Name: "days"},
			&Parameter{Type: OptionalParamType, Name: "weeks"},
		},
	}

	equal := func(param1, param2 *Parameter) bool {
		return param1.Type == param2.Type && param1.Name == param2.Name && param1.Default == param2.Default
	}

	structured := NewStructuredParser(NewHTMLNormalizer(graph))

	for i, test := range testCases {
		params := structured.parseSignature(test)
		truth := expected[i]
		require.Equal(t, len(truth), len(params), fmt.Sprintf("mismatched number of parameters for %s", test))
		for j, param := range params {
			assert.True(t, equal(param, truth[j]), fmt.Sprintf("expected: %+v, got: %+v", truth[j], param))
		}
	}
}
