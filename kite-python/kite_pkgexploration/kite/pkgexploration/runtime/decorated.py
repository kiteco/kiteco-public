""" Track underlying decorated objects at runtime; particularly useful when
    some decorators are poorly behaved (i.e. don't use functools.wraps, aren't
    signature preserving, etc)
"""
import ast
import copy
import weakref
from . import import_hook

_DECORATED = weakref.WeakKeyDictionary()  # maps objects returned by decorators to the decorated function/class


def set_decorated(decorator):
    """ Transforms decorators to additionally track the decorated object;
        useful in conjunction with qualname tracking.

        def decorated_bar():
            pass
        bar = set_decorated(arbitrary_decorator_expression)(bar)

        assert get_decorated(bar) == decorated_bar
    """
    def new_decorator(f):
        obj = decorator(f)
        # TODO(naman) really we should be using `is` here,
        # and a `WeakKeyIDDictionary` which uses key identity for hashing/equality:
        if obj != f:
            if obj not in _DECORATED:
                try:
                    _DECORATED[obj] = f
                except TypeError:
                    # certain decorators return builtins that can't be weakref'd (e.g. staticmethod)
                    pass
            # otherwise, obj was already returned by another "inner-more" decorator, so let's preserve that mapping
        return obj
    return new_decorator


def get_decorated(obj, recursive=True):
    """ Attempt to get the underlying decorated (e.g. wrapped) object; if
        recursive=True, repeatedly get the decorated object until none can be
        found.
    """
    cur = obj
    while True:
        if isinstance(cur, property):  # we don't store properties in the _DECORATED registry
            cur = cur.fget
        elif cur in _DECORATED:
            cur = _DECORATED[cur]
        else:
            break

        if not recursive:
            break

    if cur is obj:  # the loop found no decorated objects
        return None
    return cur


class RuntimeTransformer(import_hook.RuntimeTransformer):
    def visit_FunctionDef(self, node):
        self.generic_visit(node)

        # wrap all decorators in set_decorated(...)
        decorator_list = [
            self.ast_Call(self.compute_Expr(set_decorated), [d], [])
            for d in node.decorator_list
        ]

        new_node = copy.copy(node)
        new_node.decorator_list = decorator_list
        return new_node

    def visit_ClassDef(self, node):
        self.generic_visit(node)

        # wrap all decorators in set_decorated(...)
        decorator_list = [
            self.ast_Call(self.compute_Expr(set_decorated), [d], [])
            for d in node.decorator_list
        ]

        new_node = copy.copy(node)
        new_node.decorator_list = decorator_list
        return new_node
