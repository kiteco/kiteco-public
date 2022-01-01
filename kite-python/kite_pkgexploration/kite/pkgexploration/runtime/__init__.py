import sys
from kite.pkgexploration.runtime import import_hook, decorated, qualname, references

if sys.version >= '3':
    DEFAULT_TRANSFORMERS = (decorated.RuntimeTransformer, import_hook.GlobalKiteTransformer)
else:
    DEFAULT_TRANSFORMERS = (decorated.RuntimeTransformer, qualname.RuntimeTransformer, import_hook.GlobalKiteTransformer)


def patch(transformers=DEFAULT_TRANSFORMERS):
    global REFMAP
    import_hook.configure(transformers)
    REFMAP = references.configure()

    # disallow re-patching
    global patch
    patch = lambda *args, **kwargs: REFMAP

    return REFMAP
