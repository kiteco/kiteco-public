""" Automatically inject kite runtime tracking into all modules imported after configure() is called """
import ast
import copy
import collections
import imp
import os.path
import sys
import logging
import traceback

import kite

_PY3 = sys.version >= '3'


class RuntimeTransformer(ast.NodeTransformer):
    """ Base class for AST transformers; the actual transformers should be put
        in the same package as the runtime symbols referenced in the AST
    """
    @staticmethod
    def ast_Call(func, args, keywords):
        if _PY3:
            return ast.Call(func, args, keywords)
        else:
            return ast.Call(func, args, keywords, None, None)

    @staticmethod
    def compute_Expr(f, _cache={}):
        """ Get an ast expr object for the given symbol """
        if f not in _cache:
            parts = f.__module__.split('.')
            parts.append(f.__name__)

            expr = ast.Name(parts[0], ast.Load())
            for part in parts[1:]:
                expr = ast.Attribute(expr, part, ast.Load())
            _cache[f] = expr
        return _cache[f]


class GlobalKiteTransformer(RuntimeTransformer):
    """ `exec` doesn't work in the presence of free variables, so declare the
        `kite` import to be global at the top of each function. """
    def visit_FunctionDef(self, node):
        self.generic_visit(node)

        # insert the global declaration after any initial docstring
        if len(node.body) > 0 and isinstance(node.body[0], ast.Expr) and isinstance(node.body[0].value, ast.Str):
            insertion_point = 1
        else:
            insertion_point = 0

        new_body = node.body[:insertion_point]
        new_body.append(ast.Global(self.__class__.__module__.split('.', 1)[:1]))  # `global kite`
        new_body.extend(node.body[insertion_point:])

        new_node = copy.copy(node)
        new_node.body = new_body
        return new_node


if _PY3:
    import builtins
    builtins.kite = kite

    # the base class is needed; otherwise e.g. pkgutil.get_data stops working, since the Loader must implement
    # https://docs.python.org/3.7/library/importlib.html#importlib.abc.ResourceLoader.get_data
    import importlib.machinery
    class SourceLoader(importlib.machinery.SourceFileLoader):
        """ Performs the loading of a module with ast transformation """
        def __init__(self, name, spec, ast_transformers):
            super().__init__(name, spec.origin)
            self.spec = spec
            self.ast_transformers = ast_transformers

        def create_module(self, spec):
            return None

        def exec_module(self, module):
            with open(self.spec.origin, 'rb') as f:
                txt = f.read()

            # transform the ast
            tree = ast.parse(txt)
            for transformer in self.ast_transformers:
                tree = transformer().visit(tree)
            ast.fix_missing_locations(tree)

            try:
                code = compile(tree, self.spec.origin, "exec")
                exec(code, module.__dict__)
                return
            except Exception:
                # we don't put the retry in here, because then there's an explosion of how many stack traces are tracked:
                # since we track two stack traces for every nested import: 2^n stack traces.
                pass

            # retry with the original ast without tranformations
            tree = ast.parse(txt)
            code = compile(tree, self.spec.origin, "exec")
            exec(code, module.__dict__)
            # if we've gotten this far, the transformed AST failed to compile, while the original compiled
            # successfully; so log and keep going.
            logging.info("failed to compile transformed ast for module {}".format(self.spec.name))
else:
    # we must modify __builtin__ directly; if we try to put a copy of __builtin__ into the module namespace,
    # Python automatically executes in a "restricted execution mode" (https://docs.python.org/2/library/restricted.html)
    # which breaks various things.
    import __builtin__
    __builtin__.kite = kite

    class SourceLoader(object):
        """ Performs the loading of a module with ast transformation """
        def __init__(self, name, file, ast_transformers):
            self.name = name
            self.file = file
            self.ast_transformers = ast_transformers

        def load_module(self, fullname):
            txt = self.file.read()
            self.file.close()

            # transform the ast
            tree = ast.parse(txt)
            for transformer in self.ast_transformers:
                tree = transformer().visit(tree)
            ast.fix_missing_locations(tree)

            # replicate standard import machinery
            del_on_error = self.name not in sys.modules

            mod = imp.new_module(self.name)
            mod = sys.modules.setdefault(self.name, mod)

            if os.path.basename(self.file.name) == '__init__.py':  # it's a package!
                mod.__package__ = self.name
                mod.__path__ = [os.path.dirname(self.file.name)]
            else:
                mod.__package__ = self.name.rpartition('.')[0]
            mod.__file__ = self.file.name

            try:
                code = compile(tree, self.file.name, "exec")
                exec(code, mod.__dict__)
                return mod
            except Exception:
                pass

            # retry with the original ast without tranformations
            tree = ast.parse(txt)
            try:
                code = compile(tree, self.file.name, "exec")
                exec(code, mod.__dict__)
                # if we've gotten this far, the transformed AST failed to compile, while the original compiled
                # successfully; so log and keep going.
                logging.info("failed to compile transformed ast for module {}".format(self.name))
            except Exception:
                # both the transformed & original ASTs failed to compile; so clean up and raise the exception.
                if del_on_error:
                    del sys.modules[self.name]
                raise

            return mod


class ImportFinder(object):
    """ Attempt to use the standard import machinery to find a module & load it
        with ast transformations """
    def __init__(self, ast_transformers=None):
        self.ast_transformers = ast_transformers or []

    def find_spec(self, module_name, package_path, target=None):
        import importlib.machinery
        spec = importlib.machinery.PathFinder.find_spec(module_name, package_path)
        if spec is not None and spec.has_location and spec.origin.endswith('.py'):
            spec.loader = SourceLoader(module_name, spec, self.ast_transformers)
        return spec

    def find_module(self, module_name, package_path=None):
        # since we've implemented find_spec above, Python>=3.3 will always use that, so here we assume we're in Python2
        try:
            file, pathname, (suffix, mode, type) = imp.find_module(
                module_name.split('.')[-1],
                package_path
            )
            if type == imp.PY_SOURCE:
                return SourceLoader(module_name, file, self.ast_transformers)
        except ImportError as e:
            # TODO(naman) here, we probably want to defer to any other
            # meta_path finders, but still hook the loading of the source code.
            # I'm not sure how to do that yet, so in these cases we just fail
            # to inject the runtime.
            return None


def _rewrap(f):
    def wrapped(self, *args, **kwargs):
        return self.__class__(f(self, *args, **kwargs))
    return wrapped


class _TrackedList(collections.MutableSequence, list):
    # there's no reasonable way to override list + _TrackedList,
    # but override everything else
    __getitem__ = list.__getitem__
    __len__ = list.__len__
    __add__ = _rewrap(list.__add__)
    __mul__ = _rewrap(list.__mul__)
    __rmul__ = _rewrap(list.__rmul__)
    if not _PY3:
        __getslice__ = _rewrap(list.__getslice__)

    def __setitem__(self, index, value):
        if index == 0:
            raise Exception('Trying to overwrite Kite pkgexploration runtime import hook')
        list.__setitem__(self, index, value)

    def __delitem__(self, index):
        if index == 0:
            raise Exception('Trying to remove Kite pkgexploration runtime import hook')
        list.__delitem__(self, index)

    def insert(self, index, value):
        if index == 0:
            index = 1
        list.insert(self, index, value)


def configure(ast_transformers):
    """ Install import hooks to transform all subsequently imported modules
        with the passed in `ast_transformers` """
    # ensure that the Kite import hook remains the first hook
    # *unless* someone does sys.meta_path = <new list object>
    sys.meta_path = _TrackedList([ImportFinder(ast_transformers)] + sys.meta_path)
