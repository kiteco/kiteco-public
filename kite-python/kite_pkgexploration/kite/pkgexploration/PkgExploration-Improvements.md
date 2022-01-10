# PkgExploration Improvements

## Robust Top-Level Identification

Currently, we use a collection of heuristics for discovering top-level packages provided by a given Python distribution. These heuristics are fairly brittle, causing us to miss top-levels or get them slightly wrong, and this interacts poorly with namespace packages and external reference tracking. There's a better way:


1. Install the target to be explored to a separate custom location via `pip install --no-deps --target custom/ foo`. Take care to install all dependencies to the default global path. This can be achieved via `pip install foo && pip uninstall foo`.
2. Use `pkgutil.walk_packages` to find all packages/modules rooted at `custom/`. These packages will either be packages provided by `foo`, or packages under a namespace package provided by `foo`.
3. New-style namespace packages will have a `__path__`, but not a `__file__`, while old-style namespace packages can be identified by monkey patching the old-style namespace package machinery in `pkgutil` & `pkg_resources`. These packages should be ignored for the purposes of identifying top-levels (i.e. the real top-levels are under these namespaces).

## Reduce External References Within a Python Distribution

The goal of external references is to allow us to explore each Python distribution independently and link them dynamically when serving symbol information. Since each distribution cannot be broken up, it makes sense that there need not be external references between different packages in a single distribution. However, the current exploration machinery explores each top-level separately, and thus will add external references that point to different nodes provided by the same distribution. This will also allow us to avoid exploring the same node twice (once for each top-level from which it is accessible) in cases where we cannot detect external references reliably (in particular, many builtins are native, and we fail to compute a valid canonical name).

The above fix for robust package name lookup will address this issue; however, we should also keep a common object cache during exploration for all top-level packages in a given distribution and output a single large graph to avoid duplicating nodes between graphs.

Alternatively, this can be addressed as a post-processing step.

## Reproducible Builds

Currently, if we want to rebuild the docker image, this may cause a repull of the package from PyPi which can result in non-determinism in the packages that end up getting explored. We may want to rebuild to fix a bug in exploration, change the way packages are installed, etc.

To fix this, we should have a separate step of caching packages from PyPi, and then pointing pip to use our cache. This will guarantee that upstream changes don't affect our builds. This can be achieved via `pip download`, or something more fancy such as our own PyPi server.


## Include distribution in external references

When we create an external reference we lose information about the distribution that exposed the top level name for the external reference. Ideally we want to track the “distribution set” associated with each external reference (e.g the distribution that originated it along with the possible versions of that distribution).


## Include distribution in type and base references

If the type/base class comes from the current distribution we should include this, and if they are external references then we should include the distribution set (see above).


## Validate other resources that use canonical names

Arg specs, base classes, and types all use canonical names, should we be validating these during the raw graph validation process? Ideally we want to track the “distribution set” associated with each reference (e.g the distribution that originated it along with the possible versions of that distribution). Also, we probably want to explicitly explore the types associated with the parameters of a function.

## More Robust Qualname / Reference Tracking for objects

Currently, the pkgexploration kite runtime tracks statically computed names for all function and class definitions. It should be relatively straightforward to extend this to general assignments (`foo = bar()`). Since canonical paths can't currently be computed at runtime for arbitrary objects, this would dramatically improve the quality of those names. Furthermore, for these sorts of objects, the module containing the binding cannot be identified, so we should also (again, very easily) track the containing module for each binding.

This will have the nice side-effect of improving reference tracking, since those objects now get accurate names, which can be used to detect external objects. Currently, these sorts of objects get nodes in all packages that contain references to them.

## Global External Reference Validation

Currently, many external references may be dangling, since we generate each package's graph independently. We may want to consider doing a global reference validation, throwing out invalid references.

## More Robust Attribute Discovery

Currently, we use `dir()` for attribute discovery, which can be overridden by the programmer, and thus may be incorrect or incomplete (see `werkzeug/``_init__.py`). This probably can't be made foolproof, but we can use a combination of heuristics to improve the situation. One simple improvement is to add the `.__dict__.keys()` to the set of attributes to investigate.

## Node Inconsistencies in Exploration

* Orphaned nodes (i.e. nodes that aren't reachable from the root; potentially external nodes)
* Missing member/attribute nodes: nodes are added as members/attributes and are subsequently skipped in the recursive call, which causes dangling references.
* The canonical path of one node might resolve to another in certain fairly difficult-to-handle edge cases (logged as `[SEVERE]` in symbol graph generation).
* Canonical paths may have `<locals>` if defined in a function body: we should really be making the canonical path the first resolvable path to which the object is bound.
* `reflectutils.getfullname` is still broken in various other cases: this needs investigation.
* Dangling type node references may occur for dynamically generated types (where the type doesn't have a valid path under our exploration rules). This can be solved (for new-style classes) by exploring the `__class__` attribute.

## PkgExploration Failure Audit

These are all mentioned in `skipped.md` - consider that the ultimate source of truth.

### Problems With the Top-Level

* azure__2.0.0
* carbon__1.1.1
* fake-factory__9999.9.9 // top-level is Faker, but we try faker
* graphite-web__1.1.1
* ipython__6.2.1 // blacklisted (why?)
* prettytable__7
* ptyprocess__0.5.2
* pyqt5__5.10.0
* python-consul__0.7.2 // all explored names have slashes (paths)
* python-gnupg__0.4.1
* rst2pdf__0.92
* terminado__0.8.1
* testpath__0.3.1
* uritemplate.py__3.0.2
* zodb3__3.11.0  // top-level is ZODB, but we try ZODB3

### Configuration Needed

* django-haystack__2.6.1
* django-pyscss__2.0.2
* django-tables2__1.17.1

### Broken Requirements (Django)

* django-appconf__1.0.2 // made PR (accepted)
* django-autoslug__1.9.3 // made PR (pending)
* django-cors-headers__2.1.0
* django-countries__5.0
* django-filter__1.1.0
* django-fsm__2.6.0
* django-guardian__1.4.9
* django-ipware__2.0.1
* django-jsonfield__1.0.1
* django-nose__1.4.5
* django-object-actions__0.10.0
* django-picklefield__1.0.0
* django-redis-cache__1.7.1
* djangorestframework__3.7.7
* django-rest-swagger__2.1.2
* django-ses__0.8.5
* django-uuidfield__0.5.0 // requires django<1.10
* dj-static__0.0.6

### Broken Requirements (Other)

* ghostscript__0.6 // requires Ghostscript c lib (libgs)
* gitpython__2.1.8 // requires git executable
* glance_store__0.23.0 // requires netbase via eventlet
* mozcrash__1.0 // requires mozinfo
* openstackdocstheme__1.18.1 // requires sphinx
* oslo.messaging__5.35.0 // requires netbase via eventlet
* oslo.service__1.29.0 // requires netbase via eventlet
* os-win__3.0.0 // requires netbase via eventlet
* python-magic__0.4.15 // requires libmagic

### Miscellaneous

* subliminal__2.0.5 // weird NotImplementedError in pkg_resources

