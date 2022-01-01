import ast
from genshi.template.astutil import ASTCodeGenerator


class _codegenerator(ASTCodeGenerator):
    def __init__(self, tree):
        self.enabled = False
        self.line = ""
        super(_codegenerator, self).__init__(tree)

    def visit(self, tree):
        if self.line is None:
            self.line = ""
        if self.line_info is None:
            self.line_info = []
        if self.enabled:
            super(_codegenerator, self).visit(tree)


def node_to_code(node):
    gen = _codegenerator(node)
    gen.enabled = True
    gen.visit(node)
    code = gen.line
    if code.startswith("(") and code.endswith(")"):
        code = code[1:len(code)-1]
    return code
