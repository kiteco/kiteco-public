import ast

from kite.codeexamples.context import Context
from kite.codeexamples.context import node_to_type
from kite.codeexamples.context import is_literal


class ObjectUsage(object):
    def __init__(self, ident):
        self.ident = ident
        self.attributes = []

    def to_json(self):
        return {
            "Identifier": self.ident,
            "Attributes": [x.to_json() for x in self.attributes],
        }


class ObjectAttribute(object):
    def __init__(self, parent, ident, attr_type):
        self.parent = parent
        self.ident = ident
        self.attr_type = attr_type

    def to_json(self):
        return {
            "Parent": self.parent,
            "Identifier": self.ident,
            "Type": self.attr_type,
        }


class ObjectAttributeExtractor(ast.NodeVisitor):
    def __init__(self, code):
        self.context = Context()

        self.code = code
        self.func_usages = []
        self.class_usages = []
        self.usages = []
        self.attributes = []

        self._find_attributes()

    def _find_attributes(self):
        """
        Invoke ast to extract object incantations.
        """

        try:
            tree = ast.parse(self.code)
        except Exception:
            return

        self.visit(tree)

    def visit_Import(self, node):
        """ Handle imports. """

        self.context.visit_Import(node)

    def visit_ImportFrom(self, node):
        """ Handle from imports. """

        self.context.visit_ImportFrom(node)

    def visit_ClassDef(self, node):
        """
        Make sure we catch function definitions inside classes.
        """

        with self.context.class_context():
            self.generic_visit(node)

            usages = {}
            for attr in self.class_usages:
                if attr.parent not in usages:
                    usages[attr.parent] = ObjectUsage(attr.parent)
                usages[attr.parent].attributes.append(attr)

            for usage in usages.values():
                self.usages.append(usage)

            self.class_usages = []
            self.func_usages = []

    def visit_FunctionDef(self, node):
        """
        When we encounter a function, enable checking of function calls
        via self.context.in_func, so that subsequent calls to "visit_Call" via
        self.generic_visit(node) are allowed.
        """

        with self.context.function_context():
            self.generic_visit(node)

            usages = {}
            for attr in self.func_usages:
                if attr.parent not in usages:
                    usages[attr.parent] = ObjectUsage(attr.parent)
                usages[attr.parent].attributes.append(attr)

            for usage in usages.values():
                self.usages.append(usage)

            self.func_usages = []

    def visit_Assign(self, node):
        """
        Handle assignments.
        """

        self.context.visit_Assign(node)
        scopeName, resolved = self.context.resolve_node(node)
        if resolved:
            iscls = self.context.in_class_scope(node)
            self._add_name(resolved, iscls, type(node.value).__name__)

        self.generic_visit(node)

    def visit_Call(self, node):
        """
        Handle function calls called on variables that are in scope. This
        includes function args and keyword arguments
        """

        nodes = [node] + node.args + [x.value for x in node.keywords]
        for node in nodes:
            scopeName, resolved = self.context.resolve_node(node)
            if resolved:
                iscls = self.context.in_class_scope(node)
                self._add_name(resolved, iscls, type(node).__name__)

    def _add_name(self, name, iscls, attr_type):
        """
        Add a name to the list of attributes. This name is treated as a
        selector expression, with all parts except the last considered the
        parent, and the last part considered the identifier.
        """

        if name is None:
            return

        parts = name.split(".")
        parent = '.'.join(parts[:len(parts)-1])

        # This helps us ignore accidentally grabing methods and classes
        # in modules vs methods/attributes on objects.
        if parent in self.context.imports:
            return

        ident = parts[len(parts)-1]
        attr = ObjectAttribute(parent, ident, attr_type)

        self.attributes.append(attr)
        if iscls and self.context.in_class:
            self.class_usages.append(attr)
        elif self.context.in_func:
            self.func_usages.append(attr)


def get_obj_attributes(code):
    """
    get_obj_attributes takes the code, and returns a list of ObjectAttributes.
    """

    extractor = ObjectAttributeExtractor(code)
    return extractor.attributes


def get_obj_usages(code):
    """
    get_obj_usages takes the code, and returns a list of ObjectUsage objects.
    """

    extractor = ObjectAttributeExtractor(code)
    return extractor.usages
