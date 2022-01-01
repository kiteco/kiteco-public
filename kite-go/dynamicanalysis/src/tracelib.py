import types
import json
from collections import deque, OrderedDict
import macropy.core.macros
from macropy.core.macros import ast, expr
from macropy.core.hquotes import macros, u, hq, unhygienic
from macropy.core.quotes import ast as quotes_ast
# the four types below are ostensibly not used but are used by
# macro-expanded code so must be imported
from macropy.core.macros import Call, Attribute, Captured, Load

# called on each annotation if not null
callback = None

macros = macropy.core.macros.Macros()
default_output_format = "json"

types_supported = (types.FunctionType, types.BuiltinFunctionType, types.BuiltinMethodType)

# USAGE NOTES
# -----------
# invoke using `python -m run your_target_code`
#   * where your_target_code.py starts with
#       `from trace import macros, kite_trace, get_all_traced_ast_reprs`,
#       and then use `with kite_trace:` to open a block that will be traced
#   * after the code in your block has executed, call `kite_trace.get_all_traced_ast_reprs()`
#   * for maximum clarity, use get_all_traced_ast_reprs(indent='  ', include_field_names=True)
#   * for minimum clarity, use get_all_traced_ast_reprs(indent=None, include_field_names=False)   : )
#
# AST nodes have the following kwarg annotations:
#
# k_type                     will be '__kite_mixed' if the value of the expression changes across evaluations
# k_lineno and k_col_offset  as defined in https://docs.python.org/2/library/ast.html
# k_num_evals                number of times the expression was evaluated
# k_function_fqn             (FUNCTIONS ONLY) fully qualified name of function
# k_module_fqn               (MODULES ONLY) fully qualified name of module


# IMPLEMENTATION NOTES
# --------------------

# todo: annotate function defs with types of arguments for each call, num times reached, etc
# todo: annotate try/except/finally to track number of times reached,
# actual type of exceptions, etc

# note re python AST node api: the official docs aren't super descriptive about the role of each field
#   of each ast node type.  I found the following code to be very useful:
#   http://svn.edgewall.org/repos/genshi/tags/0.6.0/genshi/template/astutil.py.  Basically it prints
#   out formatted code which parses to the ast tree provided as input.  another potentially helpful url:
#   https://greentreesnakes.readthedocs.org/en/latest/

# it's also useful to look at the source code of macropy.  e.g. when working with walking the AST, see
#   https://github.com/lihaoyi/macropy/blob/13993ccb08df21a0d63b091dbaae50b9dbb3fe3e/macropy/core/walkers.py


# in get_all_traced_ast_reprs()`: all we need is a reference to the top level WrappedNodeTracker.  This
#   is trivially available when we create the top level WrappedNodeTracker, but for some odd reason this
#   doesn't work.  If you try printing the `id()` of WrappedNodeTrackers during WrappedNodeTracker.__init__()
#   vs when they're used in `wrap()`, there is NO overlap.  Macropy must be doing something funky.  So
#   we're just sticking with the strategy of awkwardly gathering the top level WrappedNodeTrackers from
#   `wrap()`, and keeping track of them through `top_level_node_trackers`.
top_level_node_trackers = set()


class Node(object):

    def __init__(self, parent, original_ast_node):
        self.parent = parent
        self.original_ast_node = original_ast_node
        self._annotations = OrderedDict()

    def annotate(self, key, value):
        if key in self._annotations and self._annotations[key] != value:
            self._annotations[key] = '__kite_mixed'
        else:
            self._annotations[key] = value

    @staticmethod
    def _get_ast_repr(ast_node, depth=0, indent='',
                      include_field_names=False, output_format=default_output_format):
        indent_text = '\n' + (indent * depth) if indent else ''

        if isinstance(ast_node, ast.AST):
            if ast_node._fields:
                field_names, field_ast_nodes = zip(*ast.iter_fields(ast_node))
                field_values = map(lambda node: NotWrappedSubtree._get_ast_repr(node,
                                                                                depth + 1, indent, include_field_names, output_format), field_ast_nodes)
            else:
                # `zip` requires at least one entry to "unzip" correctly
                field_names, field_values = [], []

            return Node._encode_ast_node_name_and_fields(ast_node, field_names, field_values, OrderedDict(
            ), depth, indent, include_field_names, output_format)

        elif isinstance(ast_node, list):
            list_values = map(lambda node: Node._get_ast_repr(node,
                                                              depth + 1, indent, include_field_names, output_format), ast_node)
            return '%s[%s%s]' % (
                indent_text, ', '.join(list_values), indent_text)

        if output_format == "json":
            return "%s%s" % (indent_text, Node._get_json_ast_repr_for_primitive(ast_node))
        return "%s%s" % (indent_text, Node._get_plain_ast_repr_for_primitive(ast_node))

    @staticmethod
    def _get_plain_ast_repr_for_primitive(ast_node):
        if isinstance(ast_node, basestring):
            return "'%s'" % (ast_node)
        return "%s" % (ast_node)

    @staticmethod
    def _get_json_ast_repr_for_primitive(ast_node):
        # add special clause for bool so it does not fall into the int case
        # below (a bool in python is also an int)
        if isinstance(ast_node, bool):
            return "\"%s\"" % (ast_node)
        elif isinstance(ast_node, int):
            return "%s" % (ast_node)
        elif isinstance(ast_node, str):
            return json.dumps(ast_node)
        elif ast_node is None:
            return "null"
        # everything else needs to be enclosed in quotes to make valid json
        return "\"%s\"" % (ast_node)

    @staticmethod
    def _encode_ast_node_name_and_fields(
            ast_node, field_names, field_values, annotations, depth, indent, include_field_names, output_format):
        annotations['k_lineno'] = ast_node.lineno
        annotations['k_col_offset'] = ast_node.col_offset

        if include_field_names:
            # start with adding fields, so that they are printed before our k_
            # annotations
            node_kw_args = OrderedDict()
            for field_name, field_value in zip(field_names, field_values):
                node_kw_args[field_name] = field_value
            node_kw_args.update(annotations)
            field_values = []
        else:
            node_kw_args = annotations

        indent_text_node = '\n' + (indent * depth) if indent else ''
        indent_text_args = '\n' + (indent * (depth + 1)) if indent else ''

        encoded = ""
        if output_format == "json":
            kw_fields_as_strings = []
            for (k, v) in node_kw_args.iteritems():
                val = str(v).strip()
                if k == "k_type" or k == "k_function_fqn" or k == "k_module_fqn" or k == "k_instance_class_fqn":
                    val = "\"%s\"" % val
                kw_fields_as_strings.append(
                    '%s\"%s\":%s' %
                    (indent_text_args, str(k).strip(), val))
            encoded = '%s{\"%s\":{%s}}' % (indent_text_node,
                                           ast_node.__class__.__name__,
                                           ', '.join(
                                               field_values + kw_fields_as_strings))
        elif output_format == "plain":
            kw_fields_as_strings = [
                '%s%s=%s' %
                (indent_text_args,
                 str(k).strip(),
                    str(v).strip()) for (
                    k,
                    v) in node_kw_args.iteritems()]
            encoded = '%s%s(%s)' % (indent_text_node,
                                    ast_node.__class__.__name__,
                                    ', '.join(
                                        field_values + kw_fields_as_strings))

        return encoded


class NotWrappedSubtree(Node):

    def get_ast_repr(self, depth=0, indent='',
                     include_field_names=False, output_format=default_output_format):
        return Node._get_ast_repr(
            self.original_ast_node, depth, indent, include_field_names, output_format)


class WrappedNodeTracker(Node):

    def __init__(self, parent, original_ast_node):
        super(WrappedNodeTracker, self).__init__(parent, original_ast_node)
        self.children = []

    @property
    def is_top_level(self):
        return self.parent is None

    @staticmethod
    def get_type_name(o):
        if not hasattr(o, '__class__'):
            return ""
        cl = o.__class__
        if hasattr(cl, '__module__'):
            module = cl.__module__
            if module is None or module == str.__class__.__module__:
                return cl.__name__
            return cl.__module__ + '.' + cl.__name__
        return cl.__name__

    def wrap(self, result):
        self.annotate('k_type', WrappedNodeTracker.get_type_name(result))

        if isinstance(result, types_supported):
            if (hasattr(result, '__module__') and result.__module__ is not None and
                    hasattr(result, '__name__') and result.__name__ is not None):
                self.annotate('k_function_fqn', result.__module__ + '.' + result.__name__)
        if isinstance(result, types.ModuleType):
            if hasattr(result, '__name__') and result.__name__ is not None:
                self.annotate('k_module_fqn', result.__name__)
        if isinstance(result, types.TypeType):
            if (hasattr(result, '__module__') and result.__module__ is not None and
                    hasattr(result, '__name__') and result.__name__ is not None):
                self.annotate('k_instance_class_fqn', result.__module__ + '.' + result.__name__)

        # skip the annotate() call because this value is supposed to change
        # over time
        if 'k_num_evals' in self._annotations:
            self._annotations['k_num_evals'] = self._annotations[
                'k_num_evals'] + 1
        else:
            self._annotations['k_num_evals'] = 1

        # go to top of tree -> gather the root WrappedNodeTracker, in preparation for
        #   `get_all_traced_ast_reprs()`
        # (see long note at top of file for detailed explanation)
        root = self
        while root.parent is not None:
            root = root.parent
        top_level_node_trackers.add(root)

        if callback is not None:
            callback()

        return result

    def get_field_values_as_strs(self, field_ast_nodes, depth=0, indent='',
                                 include_field_names=False, output_format=default_output_format):
        y = []
        ix_next_child = 0
        for ast_node in field_ast_nodes:
            if isinstance(ast_node, ast.AST) or isinstance(ast_node, list):
                # expect to have an entry in self.children for this ast_node
                y.append(
                    self.children[ix_next_child].get_ast_repr(
                        depth,
                        indent,
                        include_field_names,
                        output_format))
                ix_next_child += 1
            else:
                y.append(
                    Node._get_ast_repr(
                        ast_node,
                        depth,
                        indent,
                        include_field_names,
                        output_format))
        if ix_next_child != len(self.children):
            raise ValueError()
        if not all(map(lambda elem: isinstance(elem, str), y)):
            raise ValueError()
        return y

    def get_ast_repr(self, depth=0, indent='',
                     include_field_names=False, output_format=default_output_format):
        # inspired by `real_repr()`
        indent_text = '\n' + (indent * depth) if indent else ''
        if isinstance(self.original_ast_node, ast.AST):
            if self.original_ast_node._fields:
                field_names, field_ast_nodes = zip(
                    *ast.iter_fields(self.original_ast_node))
            else:
                # `zip` requires at least one entry to "unzip" correctly
                field_names, field_ast_nodes = [], []
            field_values = self.get_field_values_as_strs(
                field_ast_nodes,
                depth + 1,
                indent,
                include_field_names,
                output_format)

            return Node._encode_ast_node_name_and_fields(self.original_ast_node, field_names, field_values,
                                                         self._annotations, depth, indent, include_field_names, output_format)

        elif isinstance(self.original_ast_node, list):
            return '%s{"RootArray":[%s]}' % (indent_text, ', '.join(self.get_field_values_as_strs(self.original_ast_node, depth + 1, indent, include_field_names, output_format)))

        raise ValueError()

# macropy doesn't offer much state tracking while walking an AST tree, so
# we use a bit of a hack here
last_parent = None


def create_node_tracker(original_ast_node, inner):
    global last_parent
    my_parent = last_parent
    last_parent = node_tracker = WrappedNodeTracker(
        my_parent,
        original_ast_node)
    if my_parent:
        my_parent.children.append(node_tracker)
    try:
        return inner(node_tracker)
    finally:
        last_parent = my_parent

# macropy won't expose `list`s when walking along the AST (it just iterates over their elements
#   (line 74 of walkers.py)), but we'd like to create WrappedNodeTrackers for them and generally
#   approach them like AST nodes
# we achieve this by overriding the `walk_children` function of `Walker`


class KiteWalker(macropy.core.macros.Walker):

    def walk_children(self, tree, *args, **kw):
        if isinstance(tree, list):
            def inner(node_tracker):
                return super(KiteWalker, self).walk_children(tree, *args, **kw)
            return create_node_tracker(tree, inner)
        else:
            return super(KiteWalker, self).walk_children(tree, *args, **kw)


def trace_walk_func(tree, exact_src):
    @KiteWalker
    def trace_walk(tree, stop, **kw):
        def inner(node_tracker):
            # NODE TYPES WE SHOULD NOT WRAP, AND WHERE WE SHOULD IGNORE SOME FIELD(S)
            # -----------------------------------------------------------------------
            # not in the dictionary:                          wrap it, recurse on all fields
            # in the dictionary, empty fields list:     don't wrap it, recurse on all fields
            # in the dictionary, non-empty fields list: don't wrap it, recurse on all fields other than ones listed
            #
            # the first rule to match wins.  e.g. a `For` node will match the `For` entry rather than
            #   the `stmt` entry.  In general the ordering is more specific -> less specific, i.e. `stmt`
            #   is at end.
            types_to_not_wrap_and_fields_to_ignore = OrderedDict([
                # don't try to wrap left hand side (`targets`)
                (macropy.core.macros.Assign, ['targets']),
                (macropy.core.macros.AugAssign, ['target']),
                # don't try to wrap the `i` in `for i...`
                (macropy.core.macros.For, ['target']),
                # can't wrap in a function call
                (macropy.core.macros.arguments, []),
                (macropy.core.macros.excepthandler, ['name']),
                (macropy.core.macros.ClassDef, ['name', 'bases']),
                (macropy.core.macros.FunctionDef, ['name', 'args']),
                (macropy.core.macros.Delete, ['targets']),
                (macropy.core.macros.With, ['optional_vars']),
                # (all fields)
                (macropy.core.macros.Import, ['names']),
                # (all fields)
                (macropy.core.macros.ImportFrom, ['module', 'names', 'level']),
                (macropy.core.macros.Global, []),

                (macropy.core.macros.Lambda, ['args']),
                (macropy.core.macros.comprehension, ['target']),
                (macropy.core.macros.DictComp, ['key']),

                # load / store / del ... not wrappable
                (macropy.core.macros.expr_context, []),
                # you can't wrap a slice in a function call (in python
                (macropy.core.macros.slice, []),
                # you
                # can't
                # write
                # `(1,2)wrap([1])`)
                (macropy.core.macros.boolop, []),                  # and, or
                # and, sub, mult, div; e.g. you can't write `1 wrap(+) 2`
                (macropy.core.macros.operator, []),
                # invert, not, uadd, usub
                (macropy.core.macros.unaryop, []),
                # Eq, NotEq, Lt, LtE, ...
                (macropy.core.macros.cmpop, []),
                # `arg` and `value` provided as a kwarg to a function call
                (macropy.core.macros.keyword, ['arg']),
                # can't replace a statement (e.g. try/except) with a
                (macropy.core.macros.stmt, []),
                # function
                # call
            ])

            for type_to_not_wrap, fields_to_ignore in types_to_not_wrap_and_fields_to_ignore.iteritems(
            ):
                if isinstance(tree, type_to_not_wrap):
                    # there are three kinds of fields:
                    #   1) fields we are ignoring because we don't want to wrap on them -- there will be
                    #        some acrobatics to make walk_children() ignore them.
                    #   2) fields which are not ast.AST or list nodes, e.g. literals -- walk_children()
                    #        ignores these anyway but they're included in tree._fields so we have to be
                    #        aware of them.
                    #   3) fields which are are recursing on

                    all_fields_is_ast_or_list = [isinstance(value, ast.AST) or isinstance(value, list)
                                                 for field, value in ast.iter_fields(node_tracker.original_ast_node)]
                    if len(all_fields_is_ast_or_list) != len(tree._fields):
                        raise ValueError()

                    # `walk_children` (below) walks based on `tree._fields` so we're going to hack
                    #   `walk_children` to make it only walk the subset of nodes we'd like it to
                    all_fields = tree._fields
                    tree._fields = tuple(
                        filter(
                            lambda field: field not in fields_to_ignore,
                            tree._fields))  # fields to walk
                    trace_walk.walk_children(tree)  # now recurse on children
                    stop()
                    # restore _fields to all_fields / undo our hack
                    tree._fields = all_fields

                    # now we have the problem that WrappedNodeTracker expects to have children for each
                    #   ast-or-list-field, not just the subset that we walked.
                    # give it NotWrappedSubtree children for each skipped node

                    children_gathered = deque(
                        node_tracker.children)  # limited to the ones we recursed on / didn't ignore
                    node_tracker.children = []
                    for i in range(len(all_fields)):
                        if not all_fields_is_ast_or_list[i]:
                            # a literal value / similar -> no entry in
                            # `children`
                            continue

                        if all_fields[i] in fields_to_ignore:
                            node_tracker.children.append(NotWrappedSubtree(node_tracker,
                                                                           getattr(tree, all_fields[i], None)))
                        else:
                            node_tracker.children.append(
                                children_gathered.popleft())

                    return tree

            if not isinstance(tree, expr):
                raise ValueError(
                    'cannot be wrapped -> should be in types_to_not_wrap: ' + str(type(tree)))

            trace_walk.walk_children(tree)
            wrapped = hq[
                node_tracker.wrap(
                    ast[tree])]  # <- this line is where the magic happens
            stop()
            return wrapped

        return create_node_tracker(tree, inner)

    new_tree = trace_walk.recurse(tree)

    return new_tree


def get_all_traced_ast_reprs(
        indent=None, include_field_names=False, out=default_output_format):
    '''
    Returns an array of strings, one for each block or expression traced thus far in the process's lifetime.
    '''
    for top_level_node_tracker in top_level_node_trackers:
        if not top_level_node_tracker.is_top_level:
            raise ValueError()
    # .strip() below removes leading newline when indent is non-empty
    return [tracker.get_ast_repr(indent=indent, include_field_names=include_field_names, output_format=out).strip(
    ) for tracker in top_level_node_trackers]


@macros.expr
def kite_trace(tree, exact_src, **kw):
    ret = trace_walk_func(tree, exact_src)
    return ret


@macros.block
def kite_trace(tree, exact_src, **kw):
    ret = trace_walk_func(tree, exact_src)
    return ret
