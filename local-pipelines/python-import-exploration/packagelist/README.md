# General
Build a list of packages and versions to use for import exploration.

# TODO
- Separate scraping pypi from parsing info
- Full support for https://www.python.org/dev/peps/pep-0440/#version-scheme

# Notes
- This is meant for bootstrapping, in reality we should be updating the package list
  based on usage stats from user-node.
- Currently we just grab the latest version for every package that was included in the old
  import graph.
- `skipped` contains the list of packages that were potentially included in the original
  import graph that we skipped for various reasons, see file for more details.

# Package lists
- `packagelists/pip-packages` is the list of packages we tried to get pip versions for
- `packagelists/pip-versioned-packages` is the list of packages we are able to get versions for,
  this is the package list that should be consumed by the `dockertools`.
- `packagelists/special` are the packages with special installation instructions.