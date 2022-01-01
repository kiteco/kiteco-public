# Specialized parsers for python call expressions

## Updating the grammar

The grammar is in `internal/pigeon/parser.peg`. The rules with code blocks call out functions defined in `internal/pigeon/parser.go` so that static tools and editors can assist in writing the code.

Run `make clean` to remove the generated file, and `make` to generate the parser.

To make sure the generated parser is always up-to-date when the code is committed, the test `TestGeneratedParserUpToDate` will fail if the grammar was modified but the parser was not re-generated.

## Supported behavior

#### References
- https://kite.quip.com/vNE3AOhV6mmW/Robust-parsing-of-python-call-expressions
- https://kite.quip.com/kblWA0BKswab/Robust-call-parsing-spec
- https://kite.quip.com/EybcA6usL98x/Approximate-parser-improvements

The following argument types are supported for robust call parsing.

#### Attribute Expressions
we support incomplete and complete attribute expressions, incomplete attribute expressions are defined to be ones with a base and a trailing `.` but not further characters.

The following base types are supported for attribute expressions:
- Name expressions, e.g `foo.`
- Complete string literals, e.g `"foo".`
- Complete call expressions, e.g `foo().`
- Complete list, dict, or set literals.

#### Number expressions
e.g `1.2`, `1`, etc

#### Name expressions
e.g `foo`

#### Call expressions
e.g `foo()`

#### String expressions
e.g `"foo"`

We support positional arguments and keyword arguments.

#### Positional arguments
We support a subset of the possible positional argument types:
- Empty positional arguments, e.g `foo(,,,,)`
- Positional arguments that are one of:
  - Attribute expressions
  - Number expressions
  - Call expressions
  - String expressions
  - Complete List, Set, or Dict literals
  - Vararg expressions (`*args`)
  - Ellipsis (`...`)

#### Keyword arguments
We support a subset of the possible keyword argument types:
- Incomplete keyword arguments, e.g `foo(kw=)`
- Complete keyword arguments with the following values (note the name of a keyword argument must be a Name Expression):
  - Attribute expressions
  - Number expressions
  - Call expressions
  - String expressions
  - Complete List, Set, or Dict literals
  - Kwarg expressions (`**kwarg`)

#### Partial statements
We support a subset of the possible statements (first line only, no body/else/etc.):
- Class definitions
- Function definitions
- If/While/For/With statements (expressions limited to AtomExpr)

## Options
- We currently support parsing incomplete call expressions that span a maximum of `n` lines.
  - Explicit line joinings still count as a new line.
  - In the case that a call spans more than `n` lines we return a partial node including the arguments from the first `n` lines.
  - The MaxLines option can be used to specify the maximum number of lines should be included, default is 3.
- We currently support a `MaxExpressions` option which determines the total number of parse rules that the parser uses (including backtracking) when it tries to parse an input.
  - After this count is reached the parser quits and returns a nil result and an error.
  - We do not return a partial node in this case because the results are highly dependent on the particular input and thus difficult to predict.
- Clients can use the `ErrorReason` function to determine the reason why parsing failed.
  - See `ErrReason` for the supported error types.
  - The client currently supports determining if the error was caused by too many lines, in which case a partial node is returned (see above), or if the error was caused by hitting the expressions limit when attempting to parse the input (see above), all other errors return an `Unknown` error reason.
