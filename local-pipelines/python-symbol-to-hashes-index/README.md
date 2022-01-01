# General
Creates the following mappings:
- `canonical symbol` to hashes of the github files contents that contain references to the `canonical symbol` along with counts of how
  often it is referenced in particular contexts
- `symbol` to hashes of the github files contents that contain references to the `symbol` along with counts of how
  often it is referenced in particular contexts

Notes
- A `symbol` is a dotted path consisting of identifiers and attributes. In particular
  these paths have no notion of a pypi version or distribution. This means that in certain
  cases we may get results (source code or graphs) that reference paths that occur in different distributions or different versions of the same distribution. TODO: account for pypi version and distribution.
- Related: `/kiteco/kite-go/lang/python/cmds/graph-data-server`
