# TODO
- current extract pipeline uses canonical symbols, is this the right move?
- move structs into a public API?

# Known limitation
- In order to optimize throughput, the sample builder processes each seed independently and once a resulting sample is computed it
  adds the sample to the queue of finished samples and gets the next seed. This can potentially introduce a bias into
  the results since there could be cases in which for whatever reason the source code file associated with a symbol `foo`
  are all very long, this would mean that building the samples for this symbol would take a disproportionately long time
  and thus they may appear at a frequency that is lower than expected. TODO: test to make sure this does not happen? add metrics around this?

# Data types

## Symbol context
Symbol context is used to specify which context a symbol must have appeared in for it to be considered.

```
SYMBOL_CONTEXT = "symbol_context_import" | "symbol_context_attribute" | "symbol_context_name" | "symbol_context_call_func" | "symbol_context_expr" | "symbol_context_all"
```

## Distribution
Distribution of symbols, maps a symbol to a non negative weight.
```
DISTRIBUTION = traindata.SymbolDist
```

## Train data
```
TRAIN_DATA{
    expr *pythongraph.ExprTrainSample
}
```

## Samples
```
SAMPLE{
    data: TRAIN_DATA,
}
```

## Partitions
```
PARTITION{
    low: FLOAT, // in the range of [0,1)
    high: FLOAT, // in the range of (low, 1]
}
```


# Sessions

The server allows for persistent training sessions to be created. A session is configured for a distribution
of symbols and returns batches of training data which conform to the distribution.

## Creating a session

```
POST /session
```

### Request
```
{
    TODO: see sessionRequest in session.go
}
```

#### Notes
- The `interval` allows for the session to only return samples (symbol,graph) drawn from a subset of all samples. 
  If two sessions are created for the same set of symbols but with disjoint partition intervals, they will return disjoint samples.
  NOTE: we can only guarantee that the set of samples is disjoint, not that the set of hashes that are used to generate the samples are disjoint, this is because
  the same file (hash) could be used to generate multiple possible samples, one for each symbol that appears in the file.

### Response
```
{
    session: INT, // the created session ID
    samples: []SAMPLE, // samples for batch (count = batch_size)
}
```


Note that there can be attribute sites for different symbols containing the same `attribute_expr_id`, even within the same sample.
This is due to the fact that the server canonicalizes the input symbols, and so two distinct input symbols may map
to the same canonical symbol and thus share the same graph node.

## Obtaining results from an existing session

```
{
    session: INT, // the ID of the existing session
}
```

### Response

The response has the same format as the one returned for a new session.

## Keeping a session alive
```
POST /session/ping
```
This endpoint can be used to ensure that a session stays alive by periodically pinging this endpoint to ensure that
the session is not cleaned up.

### Request
```
{
    session: INT, // the ID of the existing session to ping
}
```


# Other API endpoints

## Symbol requests
The following JSON structure is used for several of the symbol request endpoints
```
SYMBOL_REQUEST{
    symbol: STRING, // symbol to return results for
    offset: INT, // offset to start returning results from
    limit: optional[INT], // maximum number of results to return (default 10)
    context: optional[SYMBOL_CONTEXT], // defaults to "symbol_context_attribute"
    canonicalize: BOOL, // canonicalize the input symbol
}
```
Notes
- Clients can determine if there are more results to return by checking if the number of
  returned results is less than limit.
- A `symbol` is a dotted path consisting of identifiers and attributes. In particular
  these paths have no notion of a pypi version or distribution. This means that in certain
  cases we may get results (source code or graphs) that reference paths that occur in different distributions or different versions of the same distribution. TODO: account for pypi version and distribution.


## Symbol Meta info
```
POST /symbol/meta-info
```
### Request
```
SYMBOL_REQUEST
```

### Response
```
{
    total_sources: INT // total number of source code files that contain the requested symbol in the specified context
}
```

## Symbol Sources
```
POST /symbol/sources
```

### Request
```
SYMBOL_REQUEST
```

### Response
```
{
    sources: []STRING // source files that contain the requested symbol in the specified context
    total: INT // total number of source files that contain the requested symbol in the specified context
}
```

## Symbol Members
```
POST /symbol/members
```

### Request
```
SYMBOL_REQUEST
```

### Response
```
{
    members: []{
        member: STRING // name of the member symbol
        score: INT // number of times the member symbol appeared in the github corpus in the specified context
    }
}
```
Notes
- This endpoint does not respect the limit or offset parameters, instead it always returns all members

## Symbol Score
```
POST /symbol/scores
```

### Request
```
{
    symbols: []STRING,
    context: optional[SYMBOL_CONTEXT], // defaults to symbol_context_attribute
    canonicalize: BOOL, // canonicalize the input symbols
}
```

### Response
```
{
    scores: map[STRING]INT // keyed by request symbol
    errors: map[STRING]string // symbols that we got an error for when getting the score
}
```
Notes
- Each symbol in the request appears in either `scores` or `errors` e.g `len(scores) + len(errors) == len(symbols)`
- Both maps are keyed by the request symbols

## Top level packages and scores
```
GET /symbol/packages
```

Returns all of the top level imports along with the count of how often they are imported in a source file.
This is not the same as the number of source files that reference the package since we count the number of time the package
was referenced in each source file.
e.g in the following file the package `abc` is counted twice
```
import abc.foo
import abc.bar
```

 ### Response
```
[]{
    name: string // name of the package
    score: INT // number of times the package is imported in all of the source files on github
}
```

 ## Imports in the source files 
```
POST /symbol/imports
```

Returns the imports in the source files that contain a reference to the symbol provided in the specified context (defaults to an attribute context), note
that depending on the symbol context provided the specified symbol may not be imported (e.g for the `symbol_context_expr`).
Note that only a subset of the files are included, you can use the `offset` field in the request to cycle through all of the posbile source files.

 ### Request
```
SYMBOL_REQUEST
```

 ### Response
```
{
    imports: [][]string // symbols imported in each file
    total: INT // the total number of source files containing the requested symbol
}
```
