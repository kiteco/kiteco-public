#!/usr/bin/env python
import unittest

from . import inference


class TestAttributeExtraction(unittest.TestCase):
    def test_extract_from_imported(self):
        code = """
import re
import numpy.random

def method():
    # This should be found
    x = re.compile()
    y = x.match()

    # This should be ignored
    z = json.dumps()
    z = z.update()

    a = numpy.random.randint()
    a.something()
"""
        expected = [
            ("re.compile", "match"),
            ("numpy.random.randint", "something"),
        ]

        self._check_expected(code, expected)

    def test_class(self):
        code = """
import re

class Foo(object):
    def __init__(self, x):
        self.x = re.compile(x)
        self.y = json.dumps(x)

    def search(self):
        y = self.x.match()
        z = self.y.update()
"""
        expected = [
            ("re.compile", "match")
        ]

        self._check_expected(code, expected)

    def test_function_call(self):
        code = """
import re

def test():
    x = re.compile()
    x.match()
"""
        expected = [
            ("re.compile", "match")
        ]

        self._check_expected(code, expected)

    def test_function_call_attributes(self):
        code = """
import re

def test():
    x = re.compile()
    foo(x.match)
"""
        expected = [
            ("re.compile", "match")
        ]

        self._check_expected(code, expected)

    def test_nested_attributes(self):
        code = """
import re

def test():
    x = re.compile()
    foo(x.match.a, x.search.b())
"""
        expected = [
            ("re.compile.match", "a"),
            ("re.compile.search", "b"),
        ]

        self._check_expected(code, expected)

    def test_literals(self):
        code = """
def test():
    a = \"hello\"
    a.upper()

    b = {1:2}
    b.update({3:4})

    c = {1,2}
    c.add(3)

    d = [1,2,3]
    d.append(4)
"""
        expected = [
            ("__builtin__.str", "upper"),
            ("__builtin__.dict", "update"),
            ("__builtin__.set", "add"),
            ("__builtin__.list", "append"),
            ("__builtin__", "dict"),
            ("__builtin__", "int"),
            ("__builtin__", "int"),
        ]

        self._check_expected(code, expected)

    def test_global_literals(self):
        code = """
a = \"hello\"
a.upper()
b = {1:2}

def test():
    b.update({3:4})
"""
        expected = [
            ("__builtin__.str", "upper"),
            ("__builtin__.dict", "update"),
            ("__builtin__", "dict")
        ]

        self._check_expected(code, expected)

    def test_ignore_changing_type(self):
        code = """
def test():
    a = \"hello\"
    a.upper()
    a = NewThing()
    a.thing()
"""
        expected = [
            ("__builtin__.str", "upper"),
        ]

        self._check_expected(code, expected)

    def test_ignore_return_call(self):
        code = """
import string

def test1():
    return string.Template()

def test2():
    tmpl = string.Template()
    return tmpl.format()

"""
        expected = [
            ("string.Template", "format"),
        ]

        self._check_expected(code, expected)

    def test_nested_assignments(self):
        code = """
import re

def test():
    a.b.c = re.compile()
    d.e.f = a.b.c
    d.e.f.match()
"""
        expected = [
            ("re.compile", "match"),
        ]

        self._check_expected(code, expected)

    def _check_expected(self, code, expected):
        attrs = inference.get_obj_attributes(code)
        self.assertEqual(
            len(expected), len(attrs),
            "expected %s got %s" % (expected, [x.to_json() for x in attrs]))
        for attr in attrs:
            self.assertIn((attr.parent, attr.ident), expected)


if __name__ == "__main__":
    unittest.main()
