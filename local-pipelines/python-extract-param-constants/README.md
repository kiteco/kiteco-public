# General
Mines the possible constants for function arguments and attribute (with string base).

For functions
- A `symbol` is a dotted path consisting of identifiers and attributes. It is a call expression.
- The info is indexed by `symbol` and `position` for positional argument and `keyword` for keyword argument.
- Only consider `int` and `string`.
- This pipeline collects the common constants that appeared more than `minConstFreq` (default is 30) times and among the top `K` (default is 10) for a given symbol and given argument, by looking at github crawled source files.

Attribute
- A map of `string -> count` for expressions look like `string.func`, for example `'\t'.join()`