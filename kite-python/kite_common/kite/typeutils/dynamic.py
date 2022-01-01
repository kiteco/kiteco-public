import types
import inspect


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
		PREFIX = "<unbound method "
		SUFFIX = ">"
		s = str(f)
		if s.startswith(PREFIX) and s.endswith(SUFFIX):
			return s[len(PREFIX):-len(SUFFIX)]

	if isinstance(f, types.MethodType):
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


def argspec(obj):
	"""
	Get a dictionary representing the call signature for the given function
	 "args" -> list of arguments, each one is a dict with keys "name" and "default_type"
	 "vararg" -> name of *arg, or None
	 "kwarg" -> name of **kwarg, or None
	"""
	try:
		spec = inspect.getargspec(obj)
	except TypeError:
		return None

	args = []
	for i, name in enumerate(spec.args):
		if not isinstance(name, basestring):
			# this can happen when args are declared as tuples, as in
			# def foo(a, (b, c)): ...
			name = "autoassigned_arg_%d" % i
		default_type = ""
		if spec.defaults is not None:
			idx = i - len(spec.args) + len(spec.defaults)
			if idx >= 0:
				default_type = fullname(getclass(spec.defaults[idx]))
		args.append(dict(name=name, default_type=default_type))
	return dict(
		args=args,
		vararg=spec.varargs,
		kwarg=spec.keywords)


def doc(x):
	"""
	Get the documentation for x, or empty string if there is no documentation.
	"""
	s = inspect.getdoc(x)
	if isinstance(s, basestring):
		return s
	else:
		return ""


def package(x):
	"""
	Get the package in which x was first defined, or return None if that cannot be determined.
	"""
	return getattr(x, "__package__", None) or getattr(x, "__module__", None)
