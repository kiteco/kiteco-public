import inspect
import logging
import re
import sys

from .runtime.qualname import get_qualname
from .runtime.decorated import get_decorated

try:  # py2/3 compatibility
    basestring
except NameError:
    basestring = (str, bytes)

try:
    import builtins
except ImportError:
    import __builtin__ as builtins

_PY3 = sys.version >= '3'
logger = logging.getLogger(__name__)


def _attr_search(container, predicate, name_hints=()):
    for iterable in (name_hints, dir(container)):
        for name in iterable:
            if name is None:  # must be a None from name_hints; ignore
                continue

            candidate = getattr(container, name, None)
            if predicate(candidate):
                return name
    raise Exception("no matching attribute found")


def _search_mro(cls, attr):
    for concrete_cls in cls.__mro__:
        if attr in concrete_cls.__dict__:
            return concrete_cls
    raise Exception("could not find attribute {} in the MRO of {}".format(attr, cls))


def get_kind(x):
    """
    Classify x as one of five high-level kinds: module, function, descriptor, type, or object
    """
    if inspect.ismodule(x):
        return "module"
    elif inspect.isroutine(
            x):  # isroutine returns true for any kind of function or method
        return "function"
    elif inspect.ismemberdescriptor(x) or inspect.isgetsetdescriptor(
            x) or inspect.isdatadescriptor(x):
        return "descriptor"
    elif inspect.isclass(x):
        return "type"
    else:
        return "object"


def get_class(obj):
    """ Get the class of an object """
    # Unfortunately for old-style classes, type(x) returns types.InstanceType.
    # But x.__class__ gives us what we want.
    return getattr(obj, "__class__", type(obj))


def boundmethod_get_classattr(bm, name_hints=()):
    """ Get the unboundmethod from a boundmethod's instance's class

    :param name_hints: list of class attribute names that might contain the desired unboundmethod
    """
    cls = get_class(bm.__self__)  # == bm.im_class for boundmethods in py2

    def search_predicate(candidate):
        if inspect.isfunction(candidate):  # in py3, unboundmethods are just functions
            candidate_fn = candidate
        elif inspect.ismethod(candidate):
            candidate_fn = candidate.__func__
        else:
            return False

        if candidate_fn is bm.__func__:
            return True
        return False

    # search for an unboundmethod on the class whose function matches the boundmethod's function
    name_hints = (bm.__name__,) + name_hints
    name = _attr_search(cls, search_predicate, name_hints=name_hints)
    try:
        concrete_cls = _search_mro(cls, name)
    except Exception:
        logger.info("boundmethod `{0}` class could not precisely be determined; continuing with a guess".format(name))
        concrete_cls = cls

    return concrete_cls, name


def approx_canonical_name(obj):
    """
    Get the fully-qualified name for the given object, or empty string if no fully qualified
    name can be deduced (which is typically the case for things that are neither types nor
    modules)
    """
    # handle builtins
    try:
        attr = str(obj)
        if getattr(builtins, attr) is obj:
            return "{}.{}".format(builtins.__name__, attr)
    except Exception:
        pass

    # handle NoneType
    if type(None) is obj:
        if _PY3:
            # there's no good fullname in Py3, so use the Py2 one
            # eventually, we should use "builtins.None.__class__"
            return "types.NoneType"
        else:
            # otherwise, we'd return __builtin__.NoneType, which is not valid
            return "types.NoneType"

    # obj is a module?
    if inspect.ismodule(obj):
        return obj.__name__

    # obj is a (decorated?) class, method, or function (py3 or Kite runtime),
    # or a generator, coroutine, or builtin (py3 only)?
    try:
        mod = inspect.getmodule(obj)
        # try to get the underlying decorated object
        underlying = get_decorated(obj)
        if underlying is None:
            underlying = obj
        return mod.__name__ + '.' + get_qualname(underlying)
    except Exception:
        pass

    # TODO we should at some point track how frequently each of these cases occurs: it might be possible to throw some of this out
    # obj is a boundmethod?
    try:
        cls, attr = boundmethod_get_classattr(obj)
        return approx_canonical_name(cls) + '.' + attr
    except Exception:
        pass

    # obj is an unboundmethod (py2)? Try this only after trying the boundmethod case!
    try:
        attr = _attr_search(obj.im_class, lambda candidate: candidate is obj, name_hints=(obj.__name__,))
        return approx_canonical_name(obj.im_class) + '.' + attr
    except Exception:
        pass

    # obj has an __objclass__?
    try:
        owner = obj.__objclass__
        if owner is not obj and not inspect.ismemberdescriptor(owner):  # TODO why do we need these checks?
            name_hints = (getattr(obj, '__name__', None),)
            attr = _attr_search(owner, lambda candidate: candidate is obj, name_hints=name_hints)
            return approx_canonical_name(owner) + '.' + attr
    except Exception:
        pass

    # obj has a __module__ and __name__?
    try:
        return obj.__module__ + '.' + obj.__name__
    except Exception:
        pass

    raise Exception("could not compute approx_canonical_name")


# # Argspec Stuff
if _PY3:
    funcsigs = inspect
else:
    import funcsigs


def _render_parameter(param):
    rendered = {
        'name': param.name,
        'default_type': None,
        'default_value': None,
        'annotation_type': None,
    }
    if param.default is not funcsigs.Parameter.empty:
        try:
            rendered['default_type'] = approx_canonical_name(get_class(param.default))
        except Exception:
            pass
        try:
            rendered['default_value'] = str(param.default)
        except Exception:
            pass
    if param.annotation is not funcsigs.Parameter.empty:
        try:
            rendered['default_annotation'] = approx_canonical_name(param.annotation),
        except Exception:
            pass
    return rendered


def get_argspec(obj):
    """ Compute an ArgSpec object for Py2/3 functions & functools.partial, returning None on exception """
    try:
        sig = funcsigs.signature(obj)
    except Exception:
        # funcsigs.signature can't handle parameter unpacking, e.g. def foo(a, (b, c)).
        # However, parameter unpacking was removed in Py3 with PEP3113,
        # and is very uncommon, so we can fail here for simplicity.
        return None

    # TODO our schema for rendering is pretty inconsistent and lossy.
    # e.g. we don't include the return annotation.
    # it might be nicer to just render the full Signature object directly
    # and reformat in post-processing based on user-node functionality
    args = []
    kwonly = []
    vararg = None
    kwarg = None
    for name in sig.parameters:
        param = sig.parameters[name]
        if param.kind == funcsigs.Parameter.VAR_POSITIONAL:
            vararg = name
        elif param.kind == funcsigs.Parameter.VAR_KEYWORD:
            kwarg = name
        elif param.kind == funcsigs.Parameter.KEYWORD_ONLY:
            kwonly.append(_render_parameter(param))
        else:
            args.append(_render_parameter(param))

    return {
        'args': args,
        'kwonly': kwonly,
        'vararg': vararg,
        'kwarg': kwarg,
    }


def get_doc(x):
    """
    Get the documentation for x, or empty string if there is no documentation.
    """
    s = inspect.getdoc(x)
    if isinstance(s, basestring):
        return s
    else:
        return ""


def get_source_info(obj):
    """ returns {'line': ..., 'path': ..., 'source': ...} containing information about an object's source """
    path = inspect.getsourcefile(obj)
    lines, line = inspect.getsourcelines(obj)

    # make it 0-indexed
    if line > 0:
        line -= 1

    return {'path': path, 'source': lines, 'line': line}
