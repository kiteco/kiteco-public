# General flow
1) The top level `signatures.Manager` calls `python.Manager.Fetch` to determine if new call data should be fetched.
2) The top level `signatures.Manager` calls `python.Manager.Handle` to get the call data for the current buffer and cursor position.

# Design decisions
- We only parse the arguments for the call that the user's cursor is over.
We do this because parsing the "Func" portion of a call expression requires a more advanced parser, in particular one
that can parse literals such as `[]` and `{}`.
SEE: https://kite.quip.com/kblWA0BKswab/Robust-call-parsing-spec

- We currently use the hash of the file contents from the start of the file to the start of the arguments to determine if we should fetch new callee data, this can be unreliable.
e.g SEE: https://github.com/kiteco/kiteco/issues/5047
TODO(juan): use the ID of the symbol associated with the value of a token to determine if we
should fetch new call data once we have a tokens based caching mechanism. We do not do this currently because a round trip to the backend is too
time intensive to ensure a good signatures experience.

- Heuristics for determining the start of the arguments for a call expression.
We currently use very simple heuristics for determing the start of the arguments for a call expression.
See: `findArgsStart` and `TestFindCallStart` for edge cases.

# References
- https://kite.quip.com/kblWA0BKswab/Robust-call-parsing-spec
- https://github.com/kiteco/kiteco/tree/master/kite-go/lang/python/pythonparser/calls/README.md
