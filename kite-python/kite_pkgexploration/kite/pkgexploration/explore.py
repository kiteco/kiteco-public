import functools
import importlib
import logging
import pkgutil
import sys

from .hooks import post_import, pre_import
from . import reflectutils

# This blacklist stinks. But there is a node inside sympy that causes the
# whole exploration to just hang unless you CTRL-C. This is pretty heavy
# handed...
# TODO(tarak): Make this blacklist more specific to the actual bad attributes
ATTR_BLACKLIST = ["closure", "__abstractmethods__"]

# This is a map from class name to kind, for objects that are supposed to be
# interpreted as having a kind other than their "actual" kind.
# see README.md: Explore Note 2
KIND_OVERRIDES = {
    "numpy.ufunc": "function",
    "werkzeug.local.LocalProxy": "module",  # TODO this isn't quite right
    "functools.partial": "function",
}

logger = logging.getLogger(__name__)


def getobjattr(obj, name):
    # if member is a (un)boundmethod, we should instead point at the underlying
    # function so that multiple attribute lookups don't generate new nodes
    # see REAME.md: Explore Note 3
    attr = getattr(obj, name)
    return getattr(attr, '__func__', attr)


def valid_canonical_name(obj):
    try:
        name = reflectutils.approx_canonical_name(obj)
    except Exception:
        return None

    if not name:
        return None

    # validate name with an import/getobjattr loop
    parts = name.split(".")
    try:
        cur = importlib.import_module(parts[0])
    except ImportError:
        return None

    for i, part in enumerate(parts):
        if i == 0:
            continue

        try:
            cur = getobjattr(cur, part)
            continue
        except AttributeError:
            pass
        try:
            # we attempt the import, but continue with getattr, not `importlib`
            # see README.md (Fullname: Note 1)
            importlib.import_module('.'.join(parts[:i + 1]))
            cur = getobjattr(cur, part)
        except (ImportError, AttributeError):
            return None

    # check that we resolved the expected object, using __eq__ instead of `is`
    # see README.md (Fullname: Note 2)
    if cur != obj:
        return None

    return name


# copied from pkgutil.walk_packages, but with extra `skipfn` functionality
# see README.md (Sub Imports: Note 2)
def walk_packages(path=None, prefix='', skipfn=None, on_error=None):
    def seen(p, m={}):
        if p in m:
            return True
        m[p] = True

    for _, name, ispkg in pkgutil.iter_modules(path, prefix):
        if skipfn and skipfn(name):
            continue

        yield name

        if ispkg:
            try:
                __import__(name)
            except ImportError:
                if on_error is not None:
                    on_error(name)
            except Exception:
                if on_error is not None:
                    on_error(name)
                else:
                    raise
            else:
                path = getattr(sys.modules[name], '__path__', None) or []

                # don't traverse path items we've seen before
                path = [p for p in path if not seen(p)]

                for subname in walk_packages(path, name + '.', skipfn, on_error):
                    yield subname


def skip_member(key):
    """
    Determine whether the given member key should be skipped for exploration.
    """
    if key == "kite":
        return True
    if key == "__init__":  # always explore __init__
        return False
    if key == "__class__":  # always explore __class__
        return False
    if key.startswith("__"):
        return True
    if key in ATTR_BLACKLIST:
        return True
    if key.startswith("func_") or key.startswith("im_"):
        return True
    return False


class Explorer(object):
    """
    Enumerates packages by recursively calling dir() on modules and classes
    """

    def __init__(self, toplevel_name, refmap=None, skipexplore=None, include_subpackages=True):
        # type: (module, ReferenceMap, Callable[[str], bool]) -> None
        self.info_by_id = {}
        self.idcache = {}
        self.toplevel_name = toplevel_name
        self.next_report = 100000
        self.refmap = refmap
        self._skipexplore = skipexplore
        self.include_subpackages = include_subpackages

    def id(self, obj):
        """
        Using this id method ensures all objects that we call id on will
        remain in memory.
        """

        ret = id(obj)
        self.idcache[ret] = obj

        if len(self.idcache) >= self.next_report:
            logger.info("Reached %d items in ID cache" % self.next_report)
            self.next_report *= 10

        return ret

    def skipexplore(self, name):
        if self._skipexplore and self._skipexplore(name):
            logger.debug("skipping {0}".format(name))
            return True
        return False

    def _get_refname(self, incoming_name, name, obj):
        """ Returns the name of the external reference for obj, and None if obj is not external """
        if self.refmap is not None:
            refname = self.refmap.lookup(incoming_name, obj)
            if refname is not None and not refname.startswith(self.toplevel_name):
                return refname

        if not name.startswith(self.toplevel_name):
            # if there is a canonical name and the path does not start with the name of the
            # package we're currently exploring, we have a reference for sure.
            return name

    def explore(self, obj, incoming_name):
        """
        Enumerate all attributes of the given object and add entries to info_by_id.
        Returns the object's id or None if exploration was skipped
        """
        if self.id(obj) in self.info_by_id or self.skipexplore(incoming_name):
            return self.id(obj)

        try:
            canonical_name = valid_canonical_name(obj)
            name = canonical_name
            if name is None:
                name = incoming_name
                logger.warning("no canonical name found for {}".format(name))

            if self.skipexplore(name):
                # don't record a node at all; this might leave a dangling node_id reference
                return

            logger.info("exploring {}".format(name))

            refname = self._get_refname(incoming_name, name, obj)
            if refname is not None:
                # external reference: record a reference node, and abort further analysis
                self.info_by_id[self.id(obj)] = {
                    'id': self.id(obj),
                    'package': self.toplevel_name,
                    'reference': refname,
                    'canonical_name': canonical_name,
                }
                return self.id(obj)

            bases = getattr(obj, "__bases__", ())

            cls = reflectutils.get_class(obj)
            # kind special casing (see README.md: Explore Note 2)
            kind = KIND_OVERRIDES.get(reflectutils.approx_canonical_name(cls), reflectutils.get_kind(obj))
            try:
                s = str(obj)
            except Exception:
                # sometimes __str__ returns non string object...
                s = ""

            # some submodules appear in the dir of the parent package, but are not found via walk_packages on the parent package
            # in which case, we should again walk the subpackages here to make sure we explore everything
            # e.g. google.cloud does not get imported through walk_packages(google) in google-cloud-bigquery, but it's available via dir(google)
            if kind == "module":
                # a module may randomly have a manually set / imported `__path__` that causes infinite recursion,
                # so we ensure that we're at a package, and not a module.
                nofile = getattr(obj, "__file__", None) is None
                pkgname = getattr(obj, "__package__", None)
                modname = getattr(obj, "__name__", None)
                # (1) we're a builtin / namespace package, which should have a __name__ but no __file__ or __package__,
                # (2) we're at an __init__.py package, which will have a __file__ and __name__ == __package__,
                # (3) we're at a module, which will have a __file__ and __name__ != __package__
                if modname is not None and (nofile or pkgname == modname): # check for cases (1)/(2)
                    self.import_subpackages(obj, modname)

            # this must happen *after* the import_subpackages above
            members = {}
            for attr in list(dir(obj)):
                if skip_member(attr):
                    logger.debug("skipping member {0} on {1}".format(attr, name))
                    continue
                try:
                    member = getobjattr(obj, attr)
                    members[attr] = member
                except BaseException as e:
                    logger.warning("failed to extract member {} from {}".format(attr, name), exc_info=True)

            node = dict(
                id=self.id(obj),
                canonical_name=canonical_name,
                str=s,
                repr=repr(obj),
                package=self.toplevel_name,
                bases=[self.id(cl) for cl in bases],
                docstring=reflectutils.get_doc(obj),
                type_id=self.id(cls),
                classification=kind,
                members={attr: self.id(members[attr]) for attr in members}
            )

            try:
                source_info = reflectutils.get_source_info(obj)
                node["source_path"] = source_info['path']
                node["source"] = source_info['source']
                node["source_begin_line"] = source_info['line']
            except Exception as e:
                logger.warning("failed to get source info for {}".format(name))

            node["argspec"] = reflectutils.get_argspec(obj)

            self.info_by_id[self.id(obj)] = node

            # # possibly recurse into node members
            if kind in ("type", "module"):
                for mname, mobj in members.items():
                    self.explore(mobj, name + "." + mname)
            else:
                # this will probably leave dangling node_id references
                logger.debug("not recursing into members of %s (classification was '%s') " % (name, kind))

            # explore the type and base classes so that any external references are processed;
            # see README.md: Explore Note 4
            self.explore(cls, name + ".__class__")
            for i, base in enumerate(bases):
                # this name is definitely not valid/navigable, but we'll fix it if necessary in post-processing
                self.explore(base, "{}.__bases__[{}]".format(name, i))

            return self.id(obj)
        except Exception as e:
            logger.warning("failed to explore {}".format(incoming_name), exc_info=True)

    def import_subpackages(self, pkg, pkg_name):
        if not self.include_subpackages:
            return

        paths = getattr(pkg, '__path__', [])

        def on_error(name):
            if self._skipexplore and self._skipexplore(name):
                return  # ignore if we wanted to skip it anyways
            # this should cause the underlying import exception to be logged
            logger.exception("failed to import {}".format(name))

        def skip_and_log(name):
            if self._skipexplore and self._skipexplore(name):
                logger.info("skipping {}".format(name))
                return True
            return False

        # use pkg_name (toplevel_name from explore_package below) here instead of pkg.__name__: see README.md (Sub Imports: Note 3)
        for name in walk_packages(getattr(pkg, '__path__', []), pkg_name + ".", skipfn=skip_and_log, on_error=on_error):
            try:
                importlib.import_module(name)
                # no need to do exploration here: see README.md (Sub Imports: Note 1)
                post_import(name)  # do post-import configuration again: see README.md (Import Hooks: Note 3)
                # TODO(naman) we probably don't need/want this
            except BaseException as e:
                logger.exception("failed to import {}".format(name))
                continue

    @classmethod
    def explore_package(cls, toplevel_name, metadata=None, include_subpackages=True, skipexplore=None, **kwargs):
        if skipexplore and skipexplore(toplevel_name):
            logger.error("skipping top-level package {0}".format(name))
            return None

        # pre- and post- import hooks: see README.md (Import Hooks)
        if metadata:
            for meta in metadata.dependencies():
                pre_import(meta.name)

        # do the import, but only for the main toplevel: see README.md (Import Hooks: Note 2)
        try:
            pkg = importlib.import_module(toplevel_name)
        except BaseException as e:
            logger.exception("failed to import {}".format(toplevel_name))
            return None

        # do post-import configuration
        # TODO handle potentially conflicting post import hooks: see README.md (Import Hooks: Note 1)
        if metadata:
            for meta in metadata.dependencies():
                post_import(meta.name)

        # init explorer for the top level package and explore
        # don't use pkg.__name__: see README.md (Explore: Note 1)
        explorer = cls(toplevel_name, skipexplore=skipexplore, include_subpackages=include_subpackages, **kwargs)

        # import subpackages to add them as attributes to the root package: see README.md (Sub Imports)
        explorer.import_subpackages(pkg, toplevel_name)

        root_id = explorer.explore(pkg, toplevel_name)

        return {
            'root_id': root_id,
            'info_by_id': explorer.info_by_id,
        }
