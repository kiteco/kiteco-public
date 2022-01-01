# Parser for numpydoc

This parses the "numpydoc" documentation convention in a docstring.

## Updating the grammar

The grammar is in `internal/pigeon/parser.peg`. The rules with code blocks call out functions defined in `internal/pigeon/parser.go` so that static tools and editors can assist in writing the code.

The PEG parser generates an `*ast.Doc` struct.

Run `make clean` to remove the generated file, and `make` to generate the parser (`DEBUG=1 make` to generate without optimizations).

To make sure the generated parser is always up-to-date when the code is committed, the test `TestGeneratedParserUpToDate` will fail if the grammar was modified but the parser was not re-generated.

## Supported formatting

The parser supports the `numpydoc` documentation convention as described in https://numpydoc.readthedocs.io/en/latest/format.html . The following sections are supported:

* Short summary
* Deprecation warning
* Extended summary
* Parameters / Attributes
* Returns / Yields
* Other parameters
* Raises
* Warns
* Warnings
* See also
* Notes
* References
* Examples
* Doctest (`>>>` lines followed by identically-indented "printed result" lines)
* List of definitions (i.e. `x : type`, possibly followed by indented paragraphs associated with the list item)
* Arbitrary sections following the same section syntax (either underline-style or directive-style)

In addition to the document structure, inline reST markup is also supported, namely:

* Italics (`*word*`)
* Bold (`**word**`)
* Monospace (double back-ticks)
* Code (single back-ticks)

The numpydoc format doesn't appear to be formally defined, so the parser is "best-effort" based on the documentation available. Per the numpydoc website:

> "While a rich set of markup is available, we limit ourselves to a very basic subset,
> in order to provide docstrings that are easy to read on text-only terminals."

#### References

The numpydoc project and documentation at https://numpydoc.readthedocs.io/en/latest/ .

Many numpydoc issues have been useful in gaining a better understanding of how the parser works, and more specifically the definition lists:

* https://github.com/numpy/numpydoc/issues/170 (the "See also" section parsing)
* https://github.com/numpy/numpydoc/issues/44 (the "Returns" section parsing, indicates that everything is treated as list of definitions)
* https://github.com/numpy/numpydoc/issues/20 (confirms the definition list parsing is driven by the section)
* https://github.com/numpy/numpydoc/issues/87 (about how to handle multi-line definition type)

Also, the reST quick reference documentation was used for inline markup parsing: http://docutils.sourceforge.net/docs/user/rst/quickref.html#inline-markup .

## Testing

To run all tests in `numpydoc` and sub-packages:

```
$ go test ./...
```
