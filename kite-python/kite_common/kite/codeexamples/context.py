import __builtin__
import types
import ast
from contextlib import contextmanager
from pprint import pprint

from utils import node_to_code


LITERAL_NODES = {
    ast.Num: "number",
    ast.Str: "str",
    ast.List: "list",
    ast.Tuple: "tuple",
    ast.Set: "set",
    ast.Dict: "dict",
}

NUMBER_TYPES = {
    int: "int",
    float: "float",
    long: "long",
    complex: "complex",
}


def is_literal(node):
    if type(node) in LITERAL_NODES:
        return True

    if isinstance(node, ast.Name):
        if node.id in ('True', 'False', 'None'):
            return True

    if type(node) is ast.Lambda:
        return True

    return False


def literal_value(node):
    if not is_literal(node):
        return ''
    if isinstance(node, ast.Num):
        return str(node.n)
    if isinstance(node, ast.Str):
        return node.s
    if (isinstance(node, ast.Name) and
            node.id in ('True', 'False', 'None')):
        return node.id
    if isinstance(node, ast.Lambda):
        return node_to_code(node)

    return ''


def _process_lambda(node):
    try:
        gen = ASTCodeGenerator()
        gen.visit(node)
        return gen.line
    except Exception as e:
        return ""


def node_to_type(node):
    if type(node) in LITERAL_NODES:
        if type(node) == ast.Num:
            for t, name in NUMBER_TYPES.items():
                if isinstance(node.n, t):
                    return name
            return LITERAL_NODES[type(node)]
        else:
            return LITERAL_NODES[type(node)]

    if type(node) == ast.Name:
        if (node.id == 'True' or node.id == 'False'):
            return "__builtin__.bool"
        elif node.id == 'None':
            return "__builtin__.None"

    if type(node) == ast.Lambda:
        return "LambdaType"

    return "unknown"


class TypeScope(object):
    def __contains__(self, name):
        return hasattr(types, name)

    def __getitem__(self, key):
        if key in self:
            return "types.%s" % key
        raise KeyError("%s is not a type" % key)


class BuiltinScope(object):
    def __contains__(self, name):
        return hasattr(__builtin__, name)

    def __getitem__(self, key):
        if key in self:
            return "__builtin__.%s" % key
        raise KeyError("%s not a builtin" % key)


class Context(object):
    def __init__(self):
        self.imports = {}
        self.func_scope = {}
        self.class_scope = {}
        self.global_scope = {}
        self.type_scope = TypeScope()
        self.builtin_scope = BuiltinScope()
        self.in_func = False
        self.in_class = False

    def resolve_node(self, node):
        name = self._get_name(node)
        if not name:
            return None, None

        scopes = [('types', self.type_scope),
                  ('builtin', self.builtin_scope),
                  ('imports', self.imports),
                  ('global', self.global_scope)]

        if self.in_func:
            scopes.append(('function', self.func_scope))
        if self.in_class:
            scopes.append(('class', self.class_scope))
        scopes.reverse()

        for scopeName, scope in scopes:
            if self._check_in(name, scope):
                return scopeName, self._resolve_in(name, scope)

        return None, None

    def in_class_scope(self, node):
        name = self._get_name(node)
        if not name:
            return False

        return self._check_in(name, self.class_scope)

    @contextmanager
    def function_context(self):
        self.start_FunctionDef()
        yield
        self.end_FunctionDef()

    @contextmanager
    def class_context(self):
        self.start_ClassDef()
        yield
        self.end_ClassDef()

    def start_FunctionDef(self):
        self.func_scope = {}
        self.in_func = True

    def end_FunctionDef(self):
        self.func_scope = {}
        self.in_func = False

    def start_ClassDef(self):
        self.class_scope = {}
        self.in_class = True

    def end_ClassDef(self):
        self.class_scope = {}
        self.in_class = False

    def visit_Import(self, node):
        """ Handle imports. """

        for im in node.names:
            if im.asname is None:
                self.imports[im.name] = im.name
            else:
                self.imports[im.asname] = im.name

    def visit_ImportFrom(self, node):
        """ Handle from imports. """

        if node.module is None:
            return

        for im in node.names:
            full_import = node.module + '.' + im.name
            if im.asname is None:
                self.imports[im.name] = full_import
            else:
                self.imports[im.asname] = full_import

    def visit_Assign(self, node):
        """
        On an assignment expression, we add variables to local or global scope
        based on whether we are in a function. We resolved the LHS and RHS to
        their full names. Then:

        - If the RHS is from an imported statement, we add it into scope bound
          to the provided LHS variable
        - If the RHS is a literal, we add it into scope bound
          to the provided LHS variable
        - Otherwise, if the LHS is already in scope, we remove it.
        """

        # Check to see if we are in a function.
        scope = self.global_scope
        if self.in_func:
            scope = self.func_scope

        # Check to see if the RHS is a known assignment type
        if not self._known_assignment_type(node.value):
            return

        # Only consider single-assignments for now
        if len(node.targets) != 1:
            return

        # Resolve left-hand and right-hand side of assignment expression
        lhs = self._get_name(node.targets[0])
        rhs = self._get_name(node.value)

        if lhs is None or rhs is None:
            return

        # Use class scope if we are in a class lhs starts with "self."
        if self.in_class and lhs.startswith("self."):
            scope = self.class_scope

        # Check imports
        if self._check_in(rhs, self.imports):
            scope[lhs] = self._resolve_in(rhs, self.imports)

        # Check current scope (global, func or class)
        elif self._check_in(rhs, scope):
            scope[lhs] = self._resolve_in(rhs, scope)

        # Check rhs in class scope
        elif self.in_class and rhs.startswith("self."):
            if self._check_in(rhs, self.class_scope):
                scope[lhs] = self._resolve_in(rhs, self.class_scope)

        # Check builtins
        elif self._check_in(rhs, self.builtin_scope):
            scope[lhs] = self._resolve_in(rhs, self.builtin_scope)

        # Check literals
        elif rhs in LITERAL_NODES.values():
            scope[lhs] = rhs

        # Remove re-assignments that were unrecognized
        elif lhs in scope:
            del scope[lhs]

    def _check_in(self, name, scope):
        if name in scope:
            return True

        parts = name.split(".")
        for i in range(1, len(parts)):
            im = '.'.join(parts[:-i])
            if im in scope:
                return True

        return False

    def _resolve_in(self, name, scope):
        if name in scope:
            return scope[name]

        parts = name.split(".")
        for i in range(1, len(parts)):
            im = '.'.join(parts[:-i])
            if im in scope:
                return scope[im] + name[len(im):]

        return None

    def _known_assignment_type(self, target):
        """
        List of types we support for the RHS in assignment expressions.
        """
        return (is_literal(target) or
                isinstance(target, (ast.Call, ast.Attribute)))

    def _get_name(self, node):
        """
        Resolve a node to its full name.
        """

        if is_literal(node):
            return node_to_type(node)

        n = node
        if isinstance(node, ast.Call):
            n = node.func

        parts = []
        while isinstance(n, ast.Attribute):
            # For function calls that are nested in selector expressions,
            # e.g os.path.join, they are chained together as a series of
            # ast.Attribute nodes. Extract them one by one.
            parts.append(n.attr)
            n = n.value

        # If we actually ended up at an ast.Name node, we have a
        # all the components of the selector expression that make up the call.
        # We just have to reverse the parts we added above.
        if isinstance(n, ast.Name):
            parts.append(n.id)
            parts.reverse()
            return '.'.join(parts)

        if isinstance(n, ast.Call):
            return self._get_name(n)

        if is_literal(n):
            nodeType = node_to_type(n)
            parts.append(nodeType)
            parts.reverse()
            return '.'.join(parts)

        return None
