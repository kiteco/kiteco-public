import re
import os
import argparse
import itertools


REPLACEMENTS = {
    '%v': '.*',
    '%+v': '.*',
    '%#v': '.*',
    '%c': '.',
    '%d': '[+-]?\d+',   # base-10 integer
    '%q': '.*',
    '%s': '.*',
    '%x': '[+-]?[0-9a-f]+'
    }

# PATTERNS from format strings that should not be escaped during conversion to regex
DO_NOT_ESCAPE = ["\\n", "\\r", "\\t"]

# Pattern for matching the filename and line number prefix
POSITION_PATTERN = '([^:]*\.go)\:(\d+)([^\d].*)?\:\ '


def regex_from_format_string(s):
    # Go through and escape non-alphanumeric chars
    # Do not use re.escape here because we want to avoid escaping "\n", "%s", etc
    regex = '^'
    pos = 0
    while pos < len(s):
        done = False
        # First consider replacing a "%x" string with the corresponding regex
        for key, replacement in REPLACEMENTS.items():
            if s[pos:].startswith(key):
                regex += '(' + replacement + ')'
                pos += len(key)
                done = True
                break
        if not done:
            # Should we escape this character?
            if s[pos].isspace():
                # whitespace should match any number of whitespace chars (including none)
                if pos > 0 and s[pos-1].isalnum():
                    regex += '\\b'  # match end of word
                regex += '\\s*'  # match any amount of whitespace
                if pos+1 < len(s) and s[pos+1].isalnum():
                    regex += '\\b'  # match end of word
            elif s[pos].isalnum() or any(s[pos:].startswith(x) for x in DO_NOT_ESCAPE):
                # do not escape this character
                regex += s[pos]
            else:
                # escape this character
                regex += '\\' + s[pos]
            pos += 1
    return regex + '$'


class ErrorContent(object):
    def __init__(self, original, pattern, text, wildcards):
        self.original = original
        self.pattern = pattern
        self.text = text
        self.wildcards = wildcards


class Pattern(object):
    def __init__(self, index, format_string):
        self.index = index
        self.format_string = format_string
        self.regex = regex_from_format_string(format_string)
        self._compiled = re.compile(self.regex)

    def match(self, s):
        m = self._compiled.match(s)
        if m is None:
            return None
        else:
            return ErrorContent(s, self, m.group(0), m.groups())


class Ontology(object):
    def __init__(self, patterns):
        # sort patterns by decreasing length so that the first match is always with the longest match
        patterns = sorted(zip(itertools.count(), patterns), key=lambda x: len(x[1]), reverse=True)
        self.patterns = [Pattern(i, p) for i, p in patterns]

    def canonicalize(self, text):
        for p in self.patterns:
            m = p.match(text)
            if m is not None:
                return m
