import os
import sys
import ast
import json
import types
import collections

import reflectutils


def getclass(obj):
    """
    Unfortunately for old-style classes, type(x) returns types.InstanceType. But x.__class__
    gives us what we want.
    """
    return getattr(obj, "__class__", type(obj))


NameEvaluation = collections.namedtuple("NameEvaluation", ["node", "value"])
AttributeEvaluation = collections.namedtuple("AttributeEvaluation", ["node", "value"])
AssignmentEvaluation = collections.namedtuple("AssignmentEvaluation", ["node", "value"])
CallEvaluation = collections.namedtuple("CallEvaluation", ["node", "func", "return_value"])


class Tracer(object):
    """
    Receives a callback each time an AST node is evaluated
    """
    def __init__(self):
        self.evaluations = []
        self.funcs_by_tag = {}
        self.func_evaluations = []

    def trace_name(self, namenode, value):
        """
        An expression was evaluated.
        """
        self.evaluations.append(NameEvaluation(namenode, value))
        return value

    def trace_attribute(self, attrnode, value):
        """
        An expression was evaluated.
        """
        self.evaluations.append(AttributeEvaluation(attrnode, value))
        return value

    def trace_assignment(self, targetnodes, value):
        """
        An assignment was evaluated. Targetnodes are the AST for the LHS. Value is the RHS.
        """
        if len(targetnodes) == 1:
            if isinstance(targetnodes[0], ast.Name):
                self.evaluations.append(AssignmentEvaluation(targetnodes[0], value))
        elif len(targetnodes) == len(value):
            for targetnode, part in zip(targetnodes, value):
                if isinstance(targetnode, ast.Name):
                    self.evaluations.append(AssignmentEvaluation(targetnode, part))
        return value

    def trace_call_func(self, node, tag, func):
        """
        A function was evaluated as part of a function call - store the result for later
        """
        self.funcs_by_tag[tag] = func
        return func

    def trace_call(self, node, tag, returnvalue):
        """
        A function was called and its return value is ready
        """
        if tag not in self.funcs_by_tag:
            sys.stderr.write("no func for tag %d\n" % tag)
            return
        func = self.funcs_by_tag[tag]
        self.evaluations.append(CallEvaluation(node, func, returnvalue))
        return returnvalue


class TraceInserter(ast.NodeTransformer):
    """
    Insert tracing nodes into an AST so that we can observe the value of each node at the point
    that it was evaluated
    """
    def __init__(self, tracer_varname):
        self.literals = {}
        self.tracer_varname = tracer_varname
        self.next_tag = 0

    def traced(self, node, funcname, *args):
        """
        Construct an AST that is functionally equivalent to NODE except that the value of
        NODE is passed at runtime to FUNCNAME.
        """
        literal_names = []
        for arg in args:
            literal_name = "_literal_%d" % len(self.literals)
            self.literals[literal_name] = arg
            literal_names.append(literal_name)

        func = ast.Attribute(
            value=ast.Name(id=self.tracer_varname, ctx=ast.Load()),
            attr=funcname,
            ctx=ast.Load())

        newnode = ast.Call(
            func=func,
            args=[ast.Name(id=lit, ctx=ast.Load()) for lit in literal_names] + [node],
            keywords=[],
            starargs=None,
            kwargs=None,
        )

        return ast.copy_location(newnode, node)

    def visit_Name(self, node):
        """
        Transform ast.Name nodes by inserting a traced node
        """
        self.generic_visit(node)
        # We must not process ast.Store nodes since those correspond to cases
        # where this node is being assigned to, as in "foo = ..."
        if isinstance(node.ctx, ast.Load):
            return self.traced(node, "trace_name", node)
        else:
            return node

    def visit_Attribute(self, node):
        """
        Transform ast.Attribute nodes by inserting a trace around the base value
        """
        self.generic_visit(node)
        return ast.Attribute(
            value=self.traced(node.value, "trace_attribute", node),
            attr=node.attr,
            ctx=node.ctx)

    def visit_Assign(self, node):
        """
        Transform ast.Assign nodes by inserting a trace around the RHS
        """
        self.generic_visit(node)
        return ast.Assign(
            targets=node.targets,
            value=self.traced(node.value, "trace_assignment", node.targets))

    def visit_Call(self, node):
        """
        Transform ast.Call nodes by inserting a trace around the function
        """
        tag = self.next_tag
        self.next_tag += 1

        self.generic_visit(node)

        call = ast.Call(
            func=self.traced(node.func, "trace_call_func", node.func, tag),
            args=node.args,
            keywords=node.keywords,
            starargs=node.starargs,
            kwargs=node.kwargs)

        return ast.copy_location(self.traced(call, "trace_call", node, tag), node)


class OffsetResolver(object):
    """
    Converts (lineno, col_offset) pairs to byte offsets from the start of a file
    """
    def __init__(self, src):
        self.cumulative = [0]
        self.src = src
        self.lines = src.split("\n")
        for line in self.lines:
            self.cumulative.append(self.cumulative[-1] + len(line) + 1)

    def byteoffset(self, lineno, col_offset):
        return self.cumulative[lineno-1] + col_offset

    def nodeoffset(self, node, default=None):
        lineno = getattr(node, "lineno", None)
        col_offset = getattr(node, "col_offset", None)
        if lineno is None or col_offset is None:
            return default
        return self.byteoffset(lineno, col_offset)


def trace(src, srcpath="src.py"):
    """
    Run the python code in SRC and return a list of (ASTNODE, VALUE) pairs representing
    the value of each expression as the time it was executed. If an expression is evaluated
    more than once (e.g. because it is inside a loop) then the same node will appear
    multiple times in the list returned from this object.
    """
    # Parse AST
    orig_tree = ast.parse(src, srcpath)

    # Transform AST
    transformer = TraceInserter("_tracer")
    tree = ast.fix_missing_locations(transformer.visit(orig_tree))

    # Compile AST to bytecode
    code = compile(tree, srcpath, "exec")

    # Construct runtime environment in which to execute transformed AST
    tracer = Tracer()
    namespace = {
        "_tracer": tracer,
        "__name__": "__main__",
    }
    namespace.update(transformer.literals)

    # Execute the compiled code
    exec code in namespace

    # Return the final evaluations
    return tracer.evaluations, orig_tree


def getfqn(value):
    instance = False
    fqn = reflectutils.fullname(value)
    if not fqn:
        instance = True
        fqn = reflectutils.fullname(getclass(value))
    return fqn, instance


def find_near_node(s, node, offsetresolver, offset=None):
    """
    Find the begin and end position of string S if it exists on the same line as NODE
    """
    line = offsetresolver.lines[node.lineno-1]
    if not offset:
        offset = offsetresolver.nodeoffset(node)
    begin = offsetresolver.src.index(s, offset)
    if begin - offset > len(line):
        raise ValueError("could not find '%s' on line '%s'", line)
    end = begin + len(s)
    return begin, end


def dotted_refs(dotted_name, begin, stem=None):
    """
    Generate references for each part in a dotted expression like "foo.bar.baz"
    """
    for i, part in enumerate(dotted_name.split(".")):
        if stem is None:
            stem = part
        else:
            stem += "." + part
        yield dict(
            begin=begin,
            end=begin+len(part),
            fully_qualified=stem,
            expression=part,
            node_type="import")
        begin += len(part) + 1


def static_references(src, syntaxtree):
    """
    Compute references from a syntax tree
    """
    offsetresolver = OffsetResolver(src)
    refs = []
    for node in ast.walk(syntaxtree):
        if isinstance(node, (ast.Import, ast.ImportFrom)):
            # Deal with an import statement
            frompkg = None
            end = offsetresolver.nodeoffset(node)
            if isinstance(node, ast.ImportFrom):
                frompkg = node.module
                begin, end = find_near_node(frompkg, node, offsetresolver, offset=end)
                refs.extend(dotted_refs(frompkg, begin))

            for alias in node.names:
                begin, end = find_near_node(alias.name, node, offsetresolver, offset=end)
                refs.extend(dotted_refs(alias.name, begin, frompkg))
                if alias.asname:
                    begin, end = find_near_node(alias.asname, node, offsetresolver, offset=end)
                    refs.append(dict(
                        begin=begin,
                        end=end,
                        fully_qualified=(frompkg+"."+alias.name if frompkg else alias.name),
                        expression=alias.asname,
                        node_type="import"))

        if isinstance(node, ast.Print):
            # Deal with a print statement
            begin, end = find_near_node("print", node, offsetresolver)
            refs.append(dict(
                begin=begin,
                end=end,
                fully_qualified="__builtin__.print",
                expression="print",
                node_type="print"))

    return refs


def dynamic_references(src, evaluations):
    """
    Given a source string and list of (ASTNODE, VALUE) pairs, return a list of references
    representing fully qualified names and the strings in the original source that refer
    to them
    """
    lines = src.split("\n")
    pos = OffsetResolver(src)

    # Collect the results
    refs = []
    for evaluation in evaluations:
        if isinstance(evaluation, (NameEvaluation, AssignmentEvaluation)):
            node, value = evaluation
            assert isinstance(node, ast.Name)

            fqn, instance = getfqn(value)
            if not fqn:
                sys.stderr.write("could not find fqn for '%s'\n" % value)
                continue

            begin = pos.nodeoffset(node)
            end = begin + len(node.id)

            refs.append(dict(
                begin=begin,
                end=end,
                expression=node.id,
                fully_qualified=fqn,
                instance=instance,
                node_type="name"))

        elif isinstance(evaluation, AttributeEvaluation):
            node, value = evaluation
            fqn, instance = getfqn(value)
            if not fqn:
                sys.stderr.write("could not find fqn for '%s'\n" % value)
                continue

            line = lines[node.lineno-1]
            rightmost = max(getattr(node, "col_offset", 0) for node in ast.walk(node.value))
            attr_col_offset = line.find("." + node.attr, rightmost+1) + 1
            if attr_col_offset == -1:
                sys.stderr.write("warning: could not find '%s' on line %d\n" % (node.attr, node.lineno))
                continue

            begin = pos.byteoffset(node.lineno, attr_col_offset)
            end = begin + len(node.attr)

            refs.append(dict(
                begin=begin,
                end=end,
                expression=node.attr,
                fully_qualified=fqn + "." + node.attr,
                instance=instance,
                node_type="attribute"))

        elif isinstance(evaluation, CallEvaluation):
            node, func, returnvalue = evaluation
            funcname = reflectutils.fullname(func)
            typename = reflectutils.fullname(getclass(returnvalue))
            if funcname is None or typename is None:
                sys.stderr.write("could not find fqn for '%s' -> '%s'\n" % (func, returnvalue))
                continue

            begin = pos.nodeoffset(node)
            end = max(pos.nodeoffset(node, 0) for node in ast.walk(node))

            refs.append(dict(
                begin=begin,
                end=end,
                expression=funcname,
                fully_qualified=typename,
                instance=instance,
                node_type="call"))

    return refs


def main():
    inpath = os.environ["SOURCE"]
    outpath = os.environ["TRACE_OUTPUT"]

    # Read the input
    with open(inpath) as f:
        src = f.read()

    # Evaluate the code
    evaluations, syntaxtree = trace(src, inpath)

    # Construct a list of references from examining the AST
    references = static_references(src, syntaxtree)

    # Construct a list of references from the evaluations
    references.extend(dynamic_references(src, evaluations))

    # Output the references
    with open(outpath, "w") as f:
        json.dump(references, f)


if __name__ == "__main__":
    main()
