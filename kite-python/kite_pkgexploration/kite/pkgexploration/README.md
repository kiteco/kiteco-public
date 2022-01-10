# pkgexploration: how it works

See `PkgExploration-Improvements.md` for future improvements.
This README describes only the current state of pkgexploration and associated notes/caveats.

The entrypoint is `Explorer.explore_packages`, which takes a top-level package name, and optionally
metadata concerning distribution dependencies. `Explorer.explore_packages` does some setup before
instantiating an `Explorer` and calling `explore` with the correct arguments.

## Import Hooks

Some distributions expect certain pieces of configuration to exist before being used (read: explored).
Notably, this includes `django` (including various extensions) and `flask`. In order to do such setup,
there are pre- and post-import hooks in `hooks.py`. In most cases post-import hooks are preferable to
pre-import hooks. Furthermore, distributions may depend on distributions that require configuration,
so we transitively collect all dependencies and run all relevant hooks.

#### Notes

1. various hooks may interfere with each other and we don't currently have a way of handling these scenarios
2. we only import the main top-level package before running post-import hooks for all transitive dependencies,
   which means that some post- hooks are actually run before the relevant package is imported.
3. we also run post-import hooks after subpackage imports, so there's an implicit assumption that
   rerunning the hook is acceptable.

## Sub Imports

We recursively import all subpackages by default (this can be disabled with `include_subpackages=False`).
The default Python behavior adds an attribute to the parent package when importing the subpackage, so we
first import all subpackages, and then explore the top-package.

#### Notes

1. Previously, we were exploring each subpackage individually in addition to importing them. This is unnecessary and
   may actually cause issues with duplication. We should just rely on Python's import behavior and walk the toplevel.
2. To do this, we use a custom/modified version of `pkgutil.walk_packages` that lives at `explore.py:walk_packages`.
   Our custom version allows providing a `skipfn` predicate for not walking specific sub-packages.
3. We were previously were using `pkg.__name__` instead of the passed-in `toplevel_name` as the prefix provided to
   `walk_packages`. This can break if a poorly written packages overrides the `__name__` lookup to return something
   inconsistent with the import path of the package (e.g. `cv2.__name__ == 'cv2.cv2'`). Since we've already imported
   `pkg` using `toplevel_name`, we should just use that as the prefix.

## Explore

When exploring a node, we do a few main things: find a traversible full/canonical name for the node, heuristically
check if the node is an external reference (i.e. imported from a package outside the one we're exploring), collect
a list of children, base classes, and type (and other metadata) for writing down, and explore the relevant nodes referenced
by the current one.

#### Notes

1. Previously, the `Explorer` class held a pointer to the toplevel `package` module, using `self.package.__name__` for
   identifying external references. See the above Sub Imports Note 3 regarding `pkg.__name__` vs `toplevel_name`.
   Since `self.package.__name__` was found to be unreliable, we should directly take a `toplevel_name` string as input,
   and pass the correct string in from `explore_packages`.
2. `KIND_OVERRIDES` lets us interpret certain objects as functions/modules as special cases. This is a hack that improves data
   quality in just a couple of isolated cases, and we should probably figure out a more general way of dealing with this.
3. In Python2, accessing method attributes of objects/classes returns special (un)boundmethod objects instead of the underlying
   function object. Notably, these objects are generated on access (and are potentially different on each access), and would
   result in multiple tracked nodes for the same "defined method." Ideally we want a single node for the single method definition,
   regardless of subclassing, etc. To do this, we always follow any `__func__` attributes, which give us a reference to the underlying
   function.
4. If we explore an object that is an instance of an externally defined type, then we want to have the `type_id` to point to an
   external reference, pointing to the full name of that type. This will happen by itself if the external type is imported somewhere
   in the package being explored (which is the common case, since it's otherwise difficult to get an instance of that type).
   However, this isn't guaranteed to occur, and in particular, doesn't occur frequently for builtin types that exist in the global
   namespace. Thus we explicitly recursively explore classes of objects to ensure that an appropriate external reference node
   is created. The same holds for base classes.

### Fullname

`reflectutils.get_fullname` uses a variety of heuristics to determine a canonical importable/traversible dotted name for a
given object. We then do additional validation (`valid_fullname`) to ensure the name resolves to the right thing.

#### Notes

1. The name should always be accessible through a sequence of `getattr`s once the relevant modules are imported. In particular,
   we should use references to the module imported via `importlib`, as some custom modules may not add themselves as attributes
   to their parents. This is the case for e.g. `tensorflow.python`, which cannot be traversed in Py3, even though it may be
   imported. This ends up not really mattering as we notice the missing attribute in post and fix the name, but it's a good
   idea to be as accurate as possible here.
2. Upon resolving the name, we use an `__eq__` equality (as opposed to `is`) check to verify we found the correct object.
    * techically we check `__ne__`, but we assume that this has the desired semantics.
    * Using `__ne__` has desirable behavior for Py2 (un)boundmethods, even though this should not be necessary as `explore`
      should recurse only on the underlying `__func__`s.
    * This also allows handling of dynamically generated attributes in well-written packages.
    * This will cause bad behavior if a poorly-written package causes more things to be equal than appopriate, which
      Naman posits is quite rare (since one would have to manually override `__ne__`). Since such poorly written
      packages can have a multitude of other effects on exploration, we accept that as an unhandled edge case.
