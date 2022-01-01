#!/usr/bin/env python
import unittest

from . import extract


class TestCuratedSnippetExtraction(unittest.TestCase):
    def test_print_call(self):
        code = """
from requests import get
print get("http://mock.kite.com/text")
"""

        expected_snippets = [
            {
                'incantations': [
                    ("requests.get", ["__builtin__.str"], {}),
                ],
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_inline_call_attribute(self):
        code = """
from requests import get
print get("http://mock.kite.com/text").encoding
"""

        expected_snippets = [
            {
                'incantations': [
                    ("requests.get", ["__builtin__.str"], {}),
                ],
                'attributes': ["requests.get"]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_attributes(self):
        code = """
import string
print string.uppercase
"""

        expected_snippets = [
            {
                'attributes': ["string.uppercase"]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def test_multiple_functions(self):
        code = """
import inspect
def f1():
    f2()

def f2():
    s = inspect.stack()
    print "===Current function==="
    print "line number:", s[0][2]
    print "function name:", s[0][3]

    print "\\n===Caller function==="
    print "line number:", s[1][2]
    print "function name:", s[1][3]

    print "\\n===Outermost call==="
    print "line number:", s[2][2]
    print "function name:", s[2][3]

f1()
"""

        expected_snippets = [
            {
                'incantations': [
                    ("inspect.stack", [], {}),
                ],
                'attributes': ["inspect.stack"]
            }
        ]

        self._check_snippets(code, expected_snippets)

    def _check_snippets(self, code, expected):
        snippets = extract.curated_snippets(code)
        self.assertEqual(
            len(expected), len(snippets),
            "expected %d, got %d (expected %s got %s)" % (len(expected), len(snippets), expected, [x.to_json() for x in snippets]))

        for idx, snippet in enumerate(snippets):
            incantations = expected[idx].get('incantations', [])
            decorators = expected[idx].get('decorators', [])
            attributes = expected[idx].get('attributes', [])

            self.assertEqual(
                len(snippet.incantations), len(incantations),
                "expected %s incantations got %s" % (incantations, [x.to_json() for x in snippet.incantations]))
            self.assertEqual(
                len(snippet.decorators), len(decorators),
                "expected %s decorators got %s" % (decorators, [x.to_json() for x in snippet.decorators]))
            self.assertEqual(
                len(snippet.attributes), len(attributes),
                "expected %s attributes got %s" % (attributes, snippet.attributes))

            self._check_incantations(incantations, snippet.incantations)
            self._check_incantations(decorators, snippet.decorators)
            for attr in attributes:
                self.assertIn(attr, snippet.attributes, "expected %s in attributes" % attr)


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
