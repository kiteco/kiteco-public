#!/usr/bin/env python
import unittest

from . import inference


class TestUsageExtraction(unittest.TestCase):
    def test_function_usage(self):
        code = """
import re
import json

def method1():
    x = re.compile()
    x.match()
    x.search()

def method2():
    x = json.dumps()
    x.update()
"""
        expected = [
            {"re.compile": ["match", "search"]},
            {"json.dumps": ["update"]},
        ]

        self._check_expected(code, expected)

    def test_class_usage(self):
        code = """
import re
import json

class Foo(object):
    def __init__(self, x):
        self.x = re.compile(x)
        self.y = json.dumps(x)

    def search(self):
        y = self.x.match()
        a = self.x.search()
        z = self.y.update()

class Bar(object):
    def __init__(self, x):
        self.x = re.compile(x)
        self.y = json.dumps(x)

    def search(self):
        y = self.x.match2()
        a = self.x.search2()
        z = self.y.update2()
"""
        expected = [
            {"re.compile": ["match", "search"]},
            {"json.dumps": ["update"]},
            {"re.compile": ["match2", "search2"]},
            {"json.dumps": ["update2"]},
        ]

        self._check_expected(code, expected)

    def test_mixed_usage(self):
        code = """
import re

def method():
    x = re.compile()
    x.match()

class Foo(object):
    def __init__(self, x):
        self.x = re.compile(x)

    def hello(self):
        self.x.search()
"""
        expected = [
            {"re.compile": ["match"]},
            {"re.compile": ["search"]}
        ]

        self._check_expected(code, expected)

    def _check_expected(self, code, expected):
        usages = inference.get_obj_usages(code)
        self.assertEqual(
            len(expected), len(usages),
            "expected %s got %s" % (
                expected,
                [{x.ident: [y.ident for y in x.attributes]} for x in usages],
            ))

if __name__ == "__main__":
    unittest.main()
