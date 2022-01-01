# Parser for argspec

This parses the "argspec" in a docstring - that is, a signature of the function described by the docstring.

## Updating the grammar

The grammar is in `internal/pigeon/parser.peg`. The rules with code blocks call out functions defined in `internal/pigeon/parser.go` so that static tools and editors can assist in writing the code.

The PEG parser generates a `*pythonimports.ArgSpec` struct.

Run `make clean` to remove the generated file, and `make` to generate the parser.

To make sure the generated parser is always up-to-date when the code is committed, the test `TestGeneratedParserUpToDate` will fail if the grammar was modified but the parser was not re-generated.

## Supported formatting

The parser supports the signatures as used in the numpy project for native functions, e.g. https://github.com/numpy/numpy/blob/master/numpy/random/mtrand/mtrand.pyx#L910 :

* The function name (an identifier) followed by parentheses
* An optional list of argument names
* Optional default values for arguments
* Vararg and Kwarg notation (i.e. `*args` and `**kwarg` arguments)
* Optional arguments delimiters (i.e. `fn(required [, optional1, optional2])` or `fn(required,/,optional)`)

#### References

The numpy project and documentation style for native functions is used as reference (https://github.com/numpy/numpy).

## Testing

To run all tests in `argspec` and sub-packages:

```
$ go test ./...
```
