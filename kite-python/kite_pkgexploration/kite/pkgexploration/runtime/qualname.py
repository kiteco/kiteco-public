""" Primitives to handle Py3 qualname emulation """
import ast
import copy
import sys
import weakref
from . import import_hook

_PY3 = sys.version >= '3'

_QUALNAMES = weakref.WeakKeyDictionary()


def get_qualname(obj):
    """ Equivalent to obj.__qualname__, but also works for
        2. dynamically generated types/classes
        3. builtins
    """
    # TODO generators should have qualnames too
    if _PY3 or hasattr(obj, '__qualname__'):
        return obj.__qualname__

    k = getattr(obj, '__func__', obj)  # get underlying function for PY2 (un)boundmethods
    if k in _QUALNAMES:
        return _QUALNAMES[k]

    return obj.__qualname__  # raise a sane exception


def set_qualname(qualname):
    """ Use as a decorator to assign the decorated function a given qualified
        name; should only be used on function/class objects:

    @set_qualname('foo.bar')
    def bar():
        pass

    assert get_qualname(bar) == 'foo.bar'
    assert kite_runtime.qualname.get_qualname(bar) == 'foo.bar'
    """
    def decorator(obj):
        try:
            k = getattr(obj, '__func__', obj)  # get underlying function for PY2 (un)boundmethods
            _QUALNAMES[k] = qualname
        except TypeError:
            pass
        return obj
    return decorator


class RuntimeTransformer(import_hook.RuntimeTransformer):
    def __init__(self):
        self.qualname_stack = []
        super(RuntimeTransformer, self).__init__()

    def visit_FunctionDef(self, node):
        self.qualname_stack.append(node.name)
        qualname = '.'.join(self.qualname_stack)
        self.qualname_stack.append('<locals>')
        self.generic_visit(node)
        self.qualname_stack.pop()  # '<locals>'
        self.qualname_stack.pop()  # node.name

        # add set_qualname(...) as the last/innermost decorator
        decorator_list = node.decorator_list + [self.ast_Call(self.compute_Expr(set_qualname), [ast.Str(qualname)], [])]

        new_node = copy.copy(node)
        new_node.decorator_list = decorator_list
        return new_node

    def visit_ClassDef(self, node):
        self.qualname_stack.append(node.name)
        qualname = '.'.join(self.qualname_stack)
        self.generic_visit(node)
        self.qualname_stack.pop()  # node.name

        # add set_qualname(...) as the last/innermost decorator
        decorator_list = node.decorator_list + [self.ast_Call(self.compute_Expr(set_qualname), [ast.Str(qualname)], [])]

        new_node = copy.copy(node)
        new_node.decorator_list = decorator_list
        return new_node
