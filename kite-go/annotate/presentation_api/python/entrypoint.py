import os
import ast
import sys
import imp

# This file is the entrypoint that that is executed when running a code example on the curation
# server. The curation backend passes the name of the file containing the code example in the
# SOURCE environment variable. This file runs that file while recording which source lines produced 
# which bytes of standard output. This is achieved by:
#   - parsing SOURCE into an AST
#   - executing each line of source using "exec"
#   - printing a special line delimieter token between each call to exec


def main():
	path = os.environ.get("SOURCE", None)
	if path is None:
		print("Required environment variable SOURCE not set")
		sys.exit(1)

	# Read source
	with open(path) as f:
		src = f.read()

	# Parse AST
	tree = ast.parse(src, path)

	# Create execution environment. Some code may rely on running inside a real module
	# and being able to find itself via sys.modules[__name__]. Some code may even be
	# hardcoded to expect __name__ == "__main__" (e.g. yaml decoding).
	namespace = imp.new_module("__main__")

	sys.modules["__kite__"] = sys.modules["__main__"]
	sys.modules["__main__"] = namespace

	# Iterate over each statement in the body of the code example
	for stmt in tree.body:
		# Find the last line in the block
		lastline = max(getattr(node, "lineno", 1) for node in ast.walk(stmt))

		# Report the line number (zero-based)
		print "[[KITE[[LINE %d]]KITE]]" % (lastline - 1)

		# Create a module that contains just this statement
		mod = ast.copy_location(ast.Module([stmt]), stmt)

		# Compile the AST to bytecode
		code = compile(mod, path, "exec")

		# Execute the code in the namespace
		exec code in namespace.__dict__


if __name__ == "__main__":
	main()
