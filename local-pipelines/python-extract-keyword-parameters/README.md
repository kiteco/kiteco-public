# General
Creates a mapping from `symbol` to a mapping from `keyword` to its frequency.

Notes
- A `symbol` is a dotted path consisting of identifiers and attributes. It is a call expression.
- A `keyword` is the name which shows up in the argument of form `name=value` in the call expression
- This pipeline creates the mapping from a keyword argument to its count for a given symbol by looking at github crawled source files.