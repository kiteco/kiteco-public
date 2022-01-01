# Parser for the epytext markup language

## Updating the grammar

The grammar is in `internal/pigeon/parser.peg`. The rules with code blocks call out functions defined in `internal/pigeon/parser.go` so that static tools and editors can assist in writing the code.

The PEG parser generates a flat list of blocks (an internal, flat "AST" found in `internal/pigeon/parser_ast.go`). This flat list is then converted to an hierarchical, traditional AST (package `ast`) on exit of the PEG parser (see `internal/pigeon/parser_ast.go`, function `toHierarchicalAST`). This is also in this translation step from internal to external AST that the inline markup is parsed (see `internal/pigeon/parser_markup.go`).

Run `make clean` to remove the generated file, and `make` to generate the parser.

To make sure the generated parser is always up-to-date when the code is committed, the test `TestGeneratedParserUpToDate` will fail if the grammar was modified but the parser was not re-generated.

## Supported formatting

The following Epytext formatting blocks are supported:

* Paragraphs
* Lists
* Sections
* Literal blocks
* Doctest Blocks
* Fields

The following Epytext inline markup is supported:

* Italicized
* Bold
* Source code
* URLs
* Documentation cross reference links (no attempt are made to resolve these links)
* Indexed terms
* Escaping

The parsed epytext AST can be rendered to HTML using the `epytext/html` package. See the [HTML rendering](#html-rendering) section for more details on this package.

#### References

- https://kite.quip.com/0rpMAiiOKXMe/Parsing-Python-Doc-Strings
- http://epydoc.sourceforge.net/epytext.html

## Verifications with epydoc

Python files in `testdata/*.py` are used to check output of the `epydoc` command-line tool on some specific docstrings.

## HTML rendering

The `epytext/html` package implements HTML rendering of an epytext AST. The rendering is implemented based on the following design decisions:

* A subset of HTML tags are used, based on Sublime Text's [mini html](https://www.sublimetext.com/docs/3/minihtml.html).
* However, fields are rendered using `<dl><dt><dd>` tags which semantically make the most sense, but Sublime Text's `minihtml` does not list those tags as officially supported.
* Sublime Text recommends adding a "plugin-specific" ID to the body element, but we opted not to add one for now (see [Best Practices](https://www.sublimetext.com/docs/3/minihtml.html#best_practices)).
* Mathematical expressions (`M` markup in epytext) are rendered as plain text.
* The summary field is included as part of the body, like any other well-known field.
* Keyword arguments and variables (instance, class and module variables) are sorted by name.
* Standard parameters are listed in the order they appear in the source.
* If multiple type definitions exist for a field, the last one is used.
* If multiple fields with the same name exist, the definition of the last one is used.

## Testing

To run all tests in `epytext` and sub-packages:

```
$ go test ./...
```

There are optional tests `TestParserDataSet` in `dataset_test.go` and `TestRenderDataSet` in `html/dataset_test.go` to run the parser/html renderer against a JSON dataset that corresponds to a Go `map[string]string` where the key is the name of the python entity and the value is the associated doc string. Because that dataset is huge and the test takes a long time to run, it is not added to the repository and an environment variable needs to be explicitly set to run the test:

* `KITE_EPYTEXT_DATASET` : required, set it to the path to the JSON file.
* `KITE_EPYTEXT_DATASET_OFFSET` : optional, the 0-based offset of the first case to run.
* `KITE_EPYTEXT_DATASET_LIMIT`: optional, the maximum number of cases to run.
* `KITE_EPYTEXT_DATASET_KEY`: optional, the name of a single case to run (the key part in the map). If set, only this case is executed.

The common core of the test that drives the testing is in function `testparser.WithDataSet` in `../internal/testparser/testparser.go`.

With a 460MB `doc-strings.json` file containing over 1.5M cases on a 2015 Macbook Pro, just unmarshaling this file takes ~7s, and running all cases would probably take around 30m (in which case make sure to set an explicit `-timeout=40m` on the `go test` command, the default being 10m).

To run on a subset of cases, the recommended way is:

```
$ KITE_EPYTEXT_DATASET=/path/to/doc-strings.json KITE_EPYTEXT_DATASET_LIMIT=100000 go test -run DataSet
```

The map of cases is sorted before execution, so the offset and limit parameters are deterministic (always run the same cases for the same values).
