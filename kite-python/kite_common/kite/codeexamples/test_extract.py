#!/usr/bin/env python
import unittest

from . import extract


class TestSnippetExtraction(unittest.TestCase):
    def test_extract_from_imported(self):
        code = """
import re
import os

def method():
    # This should be found
    x = re.compile()
    y = os.path.join("hello", "world")
    z = re.match(keyword1="value1")

    # This should be ignored
    a = weird.local.method("world")
"""
        expected_snippets = [
            {
                'incantations': [
                    ("re.compile", [], {}),
                    ("os.path.join", ["__builtin__.str", "__builtin__.str"], {}),
                    ("re.match", [], {"keyword1": "__builtin__.str"}),
                ]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_class(self):
        code = """
import os
import re

class Foo(object):
    def __init__(self, x):
        os.path.join("hello", "world")
        z = re.match(keyword1="value1")
        self.hello(what)

    def search(self):
        z = os.path.join2("hello", "world")
        re.match2(keyword1="value1")

"""

        expected_snippets = [
            {
                'incantations': [
                    ("os.path.join", ["__builtin__.str", "__builtin__.str"], {}),
                    ("re.match", [], {"keyword1": "__builtin__.str"})
                ]
            },
            {
                'incantations': [
                    ("os.path.join2", ["__builtin__.str", "__builtin__.str"], {}),
                    ("re.match2", [], {"keyword1": "__builtin__.str"}),
                ]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_nested_calls(self):
        code = """
import re
import os

def test():
    re.compile(os.path.join("hello", "world"))
"""
        expected_snippets = [
            {
                'incantations': [
                    ("os.path.join", ["__builtin__.str", "__builtin__.str"], {}),
                    ("re.compile", ["os.path.join"], {})
                ]
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_literals(self):
        code = """
import os
def test():
    os.test1(\"hello\")
    os.test2({"a": 1})
    os.test3([a, b])
"""

        expected_snippets = [
            {
                'incantations': [
                    ("os.test1", ["__builtin__.str"], {}),
                    ("os.test2", ["__builtin__.dict"], {}),
                    ("os.test3", ["__builtin__.list"], {})
                ]
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_import(self):
        code = """
import os.path

def test():
    os.test1("world")
    os.path.test1("world")
    path.test1("world")
"""
        expected_snippets = [
            {
                'incantations': [
                    ("os.path.test1", ["__builtin__.str"], {})
                ]
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_arg_resolution(self):
        code = """
import os
import argmodule
import testmodule
def test():
    v = "hello"
    os.path.join(v)
    x = argmodule.foo
    os.path.join2(x.bar)
    testmodule.write("hello".upper())
    testmodule.write2(key={a: "1"}.keys())
    z = "hello".lower()
    testmodule.write3(z)
"""

        expected_snippets = [
            {
                'incantations': [
                    ("os.path.join", ["__builtin__.str"], {}),
                    ("os.path.join2", ["argmodule.foo.bar"], {}),
                    ("testmodule.write", ["__builtin__.str.upper"], {}),
                    ("__builtin__.str.upper", [], {}),
                    ("testmodule.write2", [], {"key": "__builtin__.dict.keys"}),
                    ("__builtin__.dict.keys", [], {}),
                    ("__builtin__.str.lower", [], {}),
                    ("testmodule.write3", ["__builtin__.str.lower"], {}),
                ],
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_builtins(self):
        code = """
import testmodule

def test():
    map("a", "b")
    round(2)
    x = int(32.0)
    testmodule.test(x)
"""
        expected_snippets = [
            {
                'incantations': [
                    ("__builtin__.map", ["__builtin__.str", "__builtin__.str"], {}),
                    ("__builtin__.round", ["__builtin__.int"], {}),
                    ("__builtin__.int", ["__builtin__.float"], {}),
                    ("testmodule.test", ["__builtin__.int"], {}),
                ],
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_lambda_args(self):
        code = """
import testmodule

def test():
    testmodule.map(lambda x: x)
    testmodule.sort([], key=lambda x: x)
"""

        expected_snippets = [
            {
                'incantations': [
                    ("testmodule.map", ["types.LambdaType"], {}),
                    ("testmodule.sort", ["__builtin__.list"], { "key": "types.LambdaType" })
                ],
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_generators(self):
        code = """
def test():
    x = foo(x for (x, y) in enumerate(10))
"""
        expected_snippets = [
            {
                'incantations': [
                    ("__builtin__.enumerate", ["__builtin__.int"], {}),
                ],
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_types(self):
        code = """
def test():
    x = map(dict, [])
"""
        expected_snippets = [
            {
                'incantations': [
                    ("__builtin__.map", ["types.TypeType", "__builtin__.list"], {}),
                ],
            },
        ]

        self._check_snippets(code, expected_snippets)

    def test_decorators(self):
        code = """
import os
import testmodule

@testmodule.foo
@testmodule.bar("hello")
def test():
    return os.path.join("hello", "world")

@testmodule.baz(foo="hello")
@notimported.hello
@notimported.bar("world")
def test2():
    return os.path.join("hello", "world")
"""
        expected_snippets = [
            {
                'incantations': [
                    ("os.path.join", ["__builtin__.str", "__builtin__.str"], {})
                ],
                'decorators': [
                    ("testmodule.foo", [], {}),
                    ("testmodule.bar", ["__builtin__.str"], {})
                ]
            },
            {
                'incantations': [
                    ("os.path.join", ["__builtin__.str", "__builtin__.str"], {})
                ],
                'decorators': [
                    ("testmodule.baz", [], {'foo': '__builtin__.str'}),
                ]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def _check_snippets(self, code, expected):
        snippets = extract.get_snippets("", code)
        self.assertEqual(
            len(expected), len(snippets),
            "expected %s got %s" % (expected, [x.to_json() for x in snippets]))

        for idx, snippet in enumerate(snippets):
            incantations = expected[idx].get('incantations', [])
            decorators = expected[idx].get('decorators', [])

            self.assertEqual(
                len(snippet.incantations), len(incantations),
                "expected %s incantations got %s" % (incantations, [x.to_json() for x in snippet.incantations]))
            self.assertEqual(
                len(snippet.decorators), len(decorators),
                "expected %s decorators got %s" % (decorators, [x.to_json() for x in snippet.decorators]))

            self._check_incantations(incantations, snippet.incantations)
            self._check_incantations(decorators, snippet.decorators)

    def _check_incantations(self, expected, incantations):
            snippet_map = {}
            for inc in incantations:
                snippet_map[inc.example_of] = inc

            for name, args, kwargs in expected:
                self.assertIn(name, snippet_map,
                              "expected %s to be in %s" % (name, snippet_map))

                inc = snippet_map[name]
                self.assertEqual(len(args), inc.num_args,
                                 "expected %d args, got %d" % (len(args), inc.num_args))

                for idx in range(len(args)):
                    self.assertEqual(args[idx], inc.args[idx]['Type'],
                                     "expected %s, got %s" % (args[idx], inc.args[idx]['Type']))

                self.assertEqual(len(kwargs), inc.num_keyword_args,
                                 "expected %d args, got %d" % (len(kwargs), inc.num_keyword_args))

                for kwarg in inc.kwargs:
                    self.assertIn(kwarg['Key'], kwargs,
                                  "unexpected kwarg %s, expected one of %s" % (kwarg['Key'], kwargs))


if __name__ == "__main__":
    unittest.main()
