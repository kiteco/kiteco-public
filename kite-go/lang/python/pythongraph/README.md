# TODO
- flow names into function bodies properly
- allow variable tree in scope to handle more than just the enclosing parent function scope

# References
- https://arxiv.org/pdf/1711.00740.pdf

# Supported node types
- `ast_internal_node` -- called "syntax nodes" in the paper, corresponding to internal AST nodes.
- `ast_terminal_node` -- called "syntax tokens" in the paper, corresponding to non whitespace tokens in the lexical grammar.
- `variable_usage_node` -- called "usage nodes" in the paper, corresponding to speculative placement of variables at prediciton sites

# Supported node attributes
- `ast_node_type` -- AST node type labels (`for_stmt`, `assign_stmt`, etc), only for nodes of type `ast_internal_node`
- `literal` -- Literals for `ast_terminal_node`s.
- `types` -- Inferred types for expression nodes.
  - `NA` for nodes that do not make sense to have a type (e.g statement nodes or some terminals such as `.`).
  - `UNKNOWN_TYPE` for expression nodes that we were unable to infer the type of.

# Supported edge types
- `ast_child` edges (forward and backward) between nodes of type `ast` and `ast`
- `next_token` edges (forward and backward) between nodes of type `terminal` and `terminal`
- `ast_terminal` edges (forward and backward) between nodes of type `ast` and `terminal`
- `last_lexical_use` edges (forward and backward) between nodes of type `ast` and `ast`
- `computed_from` edges (forward and backward) between nodes of type `ast` and `ast`
- `last_read`, `last_write` edges (forward and backward) between nodes of type `ast` and `ast`
- `data_flow` -- edges (forward and backward) between nodes of type `ast` and `ast`


# Graph invariants
- Each node has exactly one incoming `next_token` edge and one outgoing `next_token` edge, the only two exceptions are
  - `EOF` node which does not have an outgoing `next_token` edge
  - `SOF` node which does not have an incoming `next_token` edge
- Nodes with type `ast_internal_node` always have an empty literal field and a non empty `ast_node_type` field
- Nodes with type `ast_terminal_node` always have a non empty literal field
