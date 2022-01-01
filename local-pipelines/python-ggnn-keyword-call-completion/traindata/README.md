# General
Creates a mapping from `symbol` to a set of call keyword arguments associated with that `symbol`.

Notes
- A `symbol` is a dotted path consisting of identifiers and attributes. Set of keyword arguments are the unique set of the name in the argument in the form `name=value` in the call expression.
- This pipeline creates this by looking at the source code that's obtained from the response of the graph-data-server. 
