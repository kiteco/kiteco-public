import inspect

try:
    import __builtin__ as builtins
    xrange
except Exception:
    import builtins
    xrange = range

BUILTIN_IMPORT = builtins.__import__

class ReferenceMap(object):
    def __init__(self):
        # importer path -> object identity -> imported path
        # this doesn't need to be a WeakRefDict, because Python import machinery caches imports anyways
        self._dict = {}

    def imp(self, *args, **kwargs):
        res = BUILTIN_IMPORT(*args, **kwargs)

        globals = args[1] if len(args) > 1 else kwargs.get('globals')
        fromlist = args[3] if len(args) > 3 else kwargs.get('fromlist')

        try:
            if fromlist and globals:
                # if globals is None, this is a manual call to __import__, which we ignore
                # otherwise, the module name should always be present
                imported_name = res.__name__
                importer_path = globals['__name__']
                for attr in fromlist:
                    try:
                        val = getattr(res, attr)
                    except AttributeError:
                        continue

                    # We can't track small integer objects without hacking CPython,
                    # because they're cached in a global array. Wrapping them up in a
                    # `WrappedInt` doesn't work because it causes interactions with native code to throw up
                    if type(val) != int or (val < -5 or 256 < val):
                        imported_path = '{}.{}'.format(imported_name, attr)
                        self._dict.setdefault(importer_path, {})[id(val)] = imported_path
        except BaseException:
            pass

        return res

    def lookup(self, importer_path, val):
        """ Get the name through which a value was imported.

        :param importer_path: every prefix of this dot-separated path is checked
                              as a namespace in which val may have been imported
        """
        if inspect.ismodule(importer_path):
            importer_path = importer_path.__name__

        parts = importer_path.split('.')
        for i in xrange(len(parts) - 1):
            path = '.'.join(parts[:len(parts) - i])
            if path in self._dict and id(val) in self._dict[path]:
                return self._dict[path][id(val)]


def configure():
    rmap = ReferenceMap()
    builtins.__import__ = rmap.imp

    return rmap
