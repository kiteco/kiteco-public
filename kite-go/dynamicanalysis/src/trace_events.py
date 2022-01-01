import os
import sys
import ast
import json
import types
import inspect
import collections


FirstObservation = collections.namedtuple("FirstObservation",
    ["id", "type_id", "canonical_name", "str", "repr", "classification", "members"])
AttributeLookup = collections.namedtuple("AttributeLookup",
    ["object_id", "attribute", "result_id"])
Call = collections.namedtuple("Call",
    ["function_id", "arguments", "vararg_id", "kwarg_id", "result_id"])


def qualname(f):
    """
    Let f = subprocess.Popen.kill
    In python 2, there is no sensible way to get the name of the class "Popen" from the object
    "f". However, str(f) in most cases looks like "<unbound method Popen.kill>". This function
    uses that fact to get the qualified name from the str method.
    """
    if hasattr(f, "__qualname__"):
        return f.__qualname__

    if isinstance(f, types.UnboundMethodType):
        # unbound methods may look like "<unbound method list.append>"
        PREFIX = "<unbound method "
        SUFFIX = ">"
        s = str(f)
        if s.startswith(PREFIX) and s.endswith(SUFFIX):
            return s[len(PREFIX):-len(SUFFIX)]

    if isinstance(f, types.MethodType):
        # bound methods may look like "<bound method set.add>"
        PREFIX = "<bound method "
        SUFFIX = " of <"
        s = str(f)
        if s.startswith(PREFIX):
            pos = s.find(SUFFIX)
            if pos != -1:
                return s[len(PREFIX):pos]

    return None


def classify(x):
    """
    Classify x as one of five high-level types: module, function, descriptor, type, or object
    """
    if inspect.ismodule(x):
        return "module"
    elif isinstance(x, (types.BuiltinFunctionType, types.FunctionType, types.MethodType, types.UnboundMethodType)):
        # Note that types.BuiltinFunctionType and types.BuitinMethodType are the same object
        return "function"
    elif type(x).__name__ in ["method_descriptor", "member_descriptor"]:
        # Unfortunately isinstance(x, types.MemberDescriptorType) does not always work!
        return "descriptor"
    elif inspect.isclass(x):
        return "type"
    else:
        return "object"


def fullname(obj):
    """
    Get the fully-qualified name for the given object, or empty string if no fully qualified
    name can be deduced (which is typically the case for things that are neither types nor 
    modules)
    """
    if inspect.ismodule(obj):
        return getattr(obj, "__name__", "")

    name = None
    if isinstance(obj, types.UnboundMethodType):
        name = qualname(obj)

    if name is None:
        name = getattr(obj, "__name__", None)

    pkg = getattr(obj, "__module__", None)
    if name is None or pkg is None:
        return None
    else:
        return pkg + "." + name


def getclass(obj):
    """
    Unfortunately for old-style classes, type(x) returns types.InstanceType. But x.__class__
    gives us what we want.
    """
    return getattr(obj, "__class__", type(obj))


class Tracer(object):
    """
    Receives a callback each time an AST node is evaluated
    """
    def __init__(self):
        self.events = []
        self.intermediates = {}
        self.seen = {}

    def last_value_of(self, node):
        # unwrap the node
        while hasattr(node, "kite_wrapped"):
            node = node.kite_wrapped
        return self.intermediates[node]

    def observe(self, value):
        """
        Add an object observation if one does not already exist, and return the object's id
        """
        if id(value) not in self.seen:
            self.seen[id(value)] = True
            fqn = fullname(value)
            typ = getclass(value)
            self.observe(typ)  # must come after self.seen[id(value)]=True
            self.events.append(FirstObservation(
                id=id(value),
                type_id=id(typ),
                canonical_name=fqn,
                str=str(value),
                repr=repr(value),
                classification=classify(value),
                members=dir(value)))
        return id(value)

    def process_attribute(self, value, node):
        """
        An attribute was evaluated
        """
        obj_value = self.last_value_of(node.value)  # object on which attribute was accessed
        self.events.append(AttributeLookup(
            object_id=self.observe(obj_value),
            attribute=node.attr,
            result_id=self.observe(value)))

    def process_call(self, value, node):
        """
        A function was called
        """
        args = []
        for arg in node.args:
            arg_value = self.last_value_of(arg)
            args.append(dict(name=None, value_id=self.observe(arg_value)))
        for arg in node.keywords:
            arg_value = self.last_value_of(arg.value)
            args.append(dict(name=arg.arg, value_id=self.observe(arg)))

        vararg_id = None
        if node.starargs:
            vararg_value = self.last_value_of(node.starargs)
            vararg_id = self.observe(vararg_value)

        kwarg_id = None
        if node.kwargs is not None:
            kwarg_value = self.last_value_of(node.kwargs)
            kwarg_id = self.observe(kwarg_value)

        func_value = self.last_value_of(node.func)

        self.events.append(Call(
            function_id=self.observe(func_value),
            arguments=args,
            vararg_id=vararg_id,
            kwarg_id=kwarg_id,
            result_id=self.observe(value)))

    def trace_expression(self, value, node):
        """
        An expression was evaluated. Targetnodes are the AST for the LHS. Value is the RHS.
        """
        self.observe(value)
        if isinstance(node, ast.Attribute):
            self.process_attribute(value, node)
        elif isinstance(node, ast.Call):
            self.process_call(value, node)
        self.intermediates[node] = value
        return value


class TraceInserter(ast.NodeTransformer):
    """
    Insert tracing nodes into an AST so that we can observe the value of each node at the point
    that it was evaluated.
    """
    def __init__(self):
        self.literals = {}
        self.tracer = Tracer()

    def literal(self, value):
        varname = "__kite_literal_%d" % len(self.literals)
        self.literals[varname] = value
        return ast.Name(id=varname, ctx=ast.Load())

    def traced(self, node, func, *args):
        """
        Construct an AST that is functionally equivalent to NODE except that the value of
        NODE is passed at runtime to FUNCNAME.
        """
        newnode = ast.Call(
            func=self.literal(func),
            args=[node] + [self.literal(arg) for arg in args],
            keywords=[],
            starargs=None,
            kwargs=None)

        newnode.kite_wrapped = node  # so that we can retrieve the original node later

        return ast.copy_location(newnode, node)

    def visit(self, node):
        # visit children first
        self.generic_visit(node)

        # ast.expr is an abstract class so we cannot just define visit_expr
        if isinstance(node, ast.expr):
            # must not transform nodes in store or del contexts
            if not hasattr(node, "ctx") or isinstance(node.ctx, ast.Load):
                node = self.traced(node, self.tracer.trace_expression, node)

        return node


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
    transformer = TraceInserter()
    tree = ast.fix_missing_locations(transformer.visit(orig_tree))

    # Compile AST to bytecode
    code = compile(tree, srcpath, "exec")

    # Construct runtime environment in which to execute transformed AST
    namespace = {
        "__name__": "__main__",
    }
    namespace.update(transformer.literals)

    # Execute the compiled code
    exec code in namespace

    # Return the final evaluations
    return transformer.tracer.events


def main():
    inpath = os.environ["SOURCE"]
    outpath = os.environ["TRACE_OUTPUT"]

    # Read the input
    with open(inpath) as f:
        src = f.read()

    # Evaluate the code
    events = trace(src, inpath)

    # Output the events
    with open(outpath, "w") as f:
        for event in events:
            typename = type(event).__name__
            json.dump(dict(type=typename, event=event._asdict()), f)


if __name__ == "__main__":
    main()
