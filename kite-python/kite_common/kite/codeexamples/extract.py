import __builtin__
import inspect
import ast
import base64
import json
import os
import sys
import zlib
import traceback
from pprint import pprint

from kite.codeexamples.context import Context
from kite.codeexamples.context import node_to_type
from kite.codeexamples.context import is_literal
from kite.codeexamples.context import literal_value


# helps us zero out values that will cause encoding errors
def encode(val):
    try:
        json.dumps([val])
        return val
    except Exception as ex:
        print >>sys.stderr, "causes error:", val, ex


class Incantation(object):
    def __init__(self):
        self.code = None
        self.example_of = None
        self.num_args = 0
        self.num_literal_args = 0
        self.num_keyword_args = 0
        self.num_literal_keyword_args = 0
        self.keywords = []
        self.has_starargs = None
        self.has_starkwargs = None
        self.nested = None
        self.lineno = None
        self.args = []
        self.kwargs = []

    def add_arg(self, varname='', valuetype='', literalval=''):
        self.args.append({
            'VarName': varname,
            'Type': valuetype,
            'Literal': encode(literalval),
        })

    def add_kwarg(self, key='', varname='', valuetype='', literalval=''):
        self.kwargs.append({
            'Key': key,
            'VarName': varname,
            'Type': valuetype,
            'Literal': encode(literalval),
        })

    def to_json(self):
        """
        This format corresponds to python.Incantation in kite-go/lang/python.
        """
        return {
            "ExampleOf": self.example_of,
            "NumArgs": self.num_args,
            "NumLiteralArgs": self.num_literal_args,
            "NumKeywordArgs": self.num_keyword_args,
            "NumLiteralKeywordArgs": self.num_literal_keyword_args,
            "HasStarArgs": self.has_starargs,
            "HasStarKwargs": self.has_starkwargs,
            "Keywords": self.keywords,
            "Nested": self.nested,
            "LineNumber": self.lineno,
            "Code": encode(self.code),
            "Args": self.args,
            "Kwargs": self.kwargs,
        }


class Snippet(object):
    def __init__(self, src_file, code, incantations, decorators, attributes):
        self.src_file = src_file
        self.incantations = incantations
        self.decorators = decorators
        self.attributes = attributes
        self.code = code
        self._compute_features()

    def _compute_features(self):
        """
        Compute basic features used to describe this snippet.
        """

        lines = self.code.split('\n')
        self.num_lines = len(lines)
        self.width = max(len(x) for x in lines)
        self.area = self.width * self.num_lines
        self.full_function = self.code.strip().startswith('def')

    def to_json(self):
        """
        This format corresponds to python.Snippet in kite-go/lang/python.
        """

        return {
            "From": self.src_file,
            "Code": encode(self.code),
            "NumLines": self.num_lines,
            "Width": self.width,
            "Area": self.area,
            "FullFunction": self.full_function,
            "Incantations": [x.to_json() for x in self.incantations],
            "Decorators": [x.to_json() for x in self.decorators],
            "Attributes": self.attributes,
            "Terms": {},
        }


class SnippetExtractor(ast.NodeVisitor):
    def __init__(self, src_file, code, curated=False):
        self.context = Context()
        self.src_file = src_file
        self.code = code
        self.lines = code.split('\n')

        self.snippets = []

        self.in_call = False
        self.cur_incantations = []
        self.cur_decorators = []
        self.cur_attributes = []
        self.cur_func_lineno = 0
        self.pending_incantations = []
        self.max_func_lineno = 0
        self.max_call_lineno = 0
        self.curated = curated

        self.compute_snippets()

    def compute_snippets(self):
        """
        Invoke ast to extract function calls.
        """

        try:
            tree = ast.parse(self.code)
        except Exception as ex:
            return

        self.visit(tree)

        # Normally, snippets are only created at function boundaries via
        # visit_FunctionDef. Need to make an exception for curated examples
        # because they (for now) aren't a part of a function.
        if self.curated:
            snip = Snippet(self.src_file,
                           self.code,
                           self.cur_incantations,
                           self.cur_decorators,
                           self.cur_attributes)
            self.snippets.append(snip)

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

    def visit_FunctionDef(self, node):
        """
        When we encounter a function, enable checking of function calls
        via self.in_func, so that subsequent calls to "visit_Call" via
        self.generic_visit(node) are allowed.
        """

        with self.context.function_context():
            self.cur_func_lineno = node.lineno

            # Don't reset if we are parsing a curated example
            if not self.curated:
                self.max_func_lineno = 0
                self.cur_incantations = []
                self.cur_decorators = []
                self.cur_attributes = []

            # Iterate over the decorator list and create incantation
            # objects for them. Decorator nodes can be ast.Name, ast.Attribute
            # or ast.Call.
            for dec in node.decorator_list:
                _, resolved = self.context.resolve_node(dec)
                if not resolved:
                    continue
                if isinstance(dec, (ast.Name, ast.Attribute)):
                    inc = Incantation()
                    inc.example_of = resolved
                    self.cur_decorators.append(inc)
                if isinstance(dec, ast.Call):
                    self.visit_Call(dec)

            for inc in self.cur_incantations:
                self.cur_decorators.append(inc)

            # Clear decorator list so we don't recurse into
            # them during generic_visit
            node.decorator_list = []

            # Clear out incantations. We have to do this because
            # visit_Call populates self.cur_incantations - and since we
            # are using it for the decorator list, we have to clear it
            # before we recurse into this node looking for more Call structures
            self.cur_incantations = []

            self.generic_visit(node)
            if (not self.curated and
                    (len(self.cur_incantations) > 0 or
                     len(self.cur_decorators) > 0 or
                     len(self.cur_attributes) > 0)):
                # Note: node.lineno is 1-based!
                code = '\n'.join(
                    self.lines[node.lineno-1:self.max_func_lineno])
                snip = Snippet(self.src_file,
                               code,
                               self.cur_incantations,
                               self.cur_decorators,
                               self.cur_attributes)
                self.snippets.append(snip)

            # Don't reset if we are parsing a curated example.
            if not self.curated:
                self.cur_incantations = []
                self.cur_decorators = []
                self.cur_attributes = []

    def visit_Assign(self, node):
        """
        Make sure we keep track of context updates via assignments.
        """

        self.generic_visit(node)
        self.context.visit_Assign(node)

    def visit(self, node):
        """
        This method overrides ast.NodeVisitor's visit method to intercept
        nodes that are visited to find the maximum line number contains within
        a subset of visit calls (controlled by self.context.in_func). The goal
        is to determine the line number for where the function definition ends.
        It is used in visit_FunctionDef to extract the code of the function
        definition.
        """

        if hasattr(node, 'lineno'):
            if self.context.in_func and node.lineno > self.max_func_lineno:
                self.max_func_lineno = node.lineno
            if self.in_call and node.lineno > self.max_call_lineno:
                self.max_call_lineno = node.lineno

        return super(SnippetExtractor, self).visit(node)

    def visit_Call(self, node):
        """
        Implements ast.NodeVisitor method that is called
        when visitor encounters a Call object in the AST.
        """

        if not self.curated and not self.context.in_func:
            return

        _, resolved = self.context.resolve_node(node)
        if not resolved:
            self.generic_visit(node)
            return

        # Build the incantation..
        inc = Incantation()
        inc.example_of = resolved
        inc.num_args = len(node.args)
        inc.num_keyword_args = len(node.keywords)
        inc.has_starargs = (node.starargs is not None)
        inc.has_starkwargs = (node.kwargs is not None)
        inc.keywords = [x.arg for x in node.keywords]
        inc.num_literal_args = len([x for x in node.args if is_literal(x)])
        inc.num_literal_keyword_args = len(
            [x for x in node.keywords if is_literal(x.value)])
        inc.nested = self.in_call
        inc.lineno = node.lineno - self.cur_func_lineno

        # BEGIN EPIC HACK
        # This is here because this parser doesn't really do any
        # type resolution, but we want to distinguish between cases
        # such as map(str, [1,2]) and foo("hello", [1,2]). Basically,
        # of the argument points directly to a builtin, use that
        # builtin's type
        def fake_resolve_builtin(scope, name, resolved, literal):
            if scope == "builtin" and "__builtin__."+name == resolved:
                attr = getattr(__builtin__, name)
                if inspect.isclass(attr):
                    resolved = "types.TypeType"
                elif inspect.isroutine(attr):
                    resolved = "types.BuiltinFunctionType"
                else:
                    resolved = ""
                literal = name
                name = ""
            if scope == "types" and "types."+name == resolved:
                resolved = "types.Type"
            return name, resolved, literal

        for k in node.keywords:
            name = ''
            literal = literal_value(k.value)
            if isinstance(k.value, ast.Name):
                name = k.value.id
            scope, resolved = self.context.resolve_node(k.value)
            if not resolved:
                resolved = ""

            name, resolved, literal = fake_resolve_builtin(
                scope, name, resolved, literal)
            inc.add_kwarg(key=k.arg,
                          varname=name,
                          valuetype=resolved,
                          literalval=literal)

        for a in node.args:
            name = ''
            literal = literal_value(a)
            if isinstance(a, ast.Name):
                name = a.id
            scope, resolved = self.context.resolve_node(a)
            if not resolved:
                resolved = ""

            name, resolved, literal = fake_resolve_builtin(
                scope, name, resolved, literal)
            inc.add_arg(varname=name,
                        valuetype=resolved,
                        literalval=literal)

        self.pending_incantations.append(inc)

        was_false = not self.in_call
        self.in_call = True

        self.generic_visit(node)

        if was_false:
            self.in_call = False
            code = '\n'.join(self.lines[node.lineno-1:self.max_call_lineno])
            for inc in self.pending_incantations:
                inc.code = code
                self.cur_incantations.append(inc)
            self.pending_incantations = []

    def visit_Attribute(self, node):
        """
        Collect attributes that we encounter.
        """

        self.generic_visit(node)
        if not self.curated and not self.context.in_func:
            return

        _, resolved = self.context.resolve_node(node)
        if resolved:
            self.cur_attributes.append(resolved)


def curated_snippets(code):
    """
    curated_snippet takes curated code and
    returns a parsed snippet containing that example.
    """

    extractor = SnippetExtractor("curated", code, curated=True)
    return extractor.snippets


def get_snippets(path, code):
    """
    extract takes the code, and path (original URI), and returns a
    list of Snippet objects.
    """

    extractor = SnippetExtractor(path, code)
    return extractor.snippets
