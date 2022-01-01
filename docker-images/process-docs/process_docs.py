from __future__ import print_function
import sys
import json
from docutils import core


def html_string(input_string):
    overrides = {'input_encoding': 'unicode',
                 'doctitle_xform': 1,
                 'initial_header_level': 1,
                 'report_level': 5,  # suppress errors for unresolved roles and directives
                 'halt_level': 5  # never halt on an error
                 }
    return core.publish_string(source=input_string, writer_name='html', settings_overrides=overrides)


def trim_leading_spaces(text):
    """Remove unnecessary indent from each line in the text.
    """
    modified = ""
    min_spaces = sys.maxsize
    split = text.splitlines()
    # count indentation of each line and track minimum (if positive, reflects
    # unnecessary indentation in text)
    for line in split[1:]:
        if len(line) == 0:
            continue
        spaces = len(line) - len(line.lstrip(' '))
        if spaces < min_spaces:
            min_spaces = spaces

    if min_spaces == sys.maxsize:
        min_spaces = 0

    # shift each line to the left by minimum
    modified = text
    if min_spaces > 0:
        l = []
        for i, line in enumerate(split):
            if i == 0 or len(line) == 0:
                l.append(line)
                continue
            l.append(line[min_spaces:])
        if len(l) > 0:
            modified = ''.join(l)

    return modified


def process_raw_docstring():
    for line in sys.stdin: 
        try:
            json_data = json.loads(line, strict=False)
        except:
            print("can't process stdin", line, file=sys.stderr)
            json_data = {}

        html = ""
        if "docstring" in json_data:
            clean = trim_leading_spaces(json_data["docstring"])  # docutils complains if incorrectly indented
            clean = u".. role:: func\n.. role:: class\n.. role:: meth\n.. role:: mod\n.. role:: attr\n.. role:: data\n.. role:: const\n.. role:: exc\n.. role:: obj\n.. role:: ref\n" + clean
            html = html_string(clean)  # docutils conversion
            html = html.decode()  # utf-8 to unicode
        converted = {
                "identifier": json_data.get("identifier", ""),
                "html": html
        }
        print(json.dumps(converted))
        sys.stdout.flush()


if __name__ == "__main__":
    process_raw_docstring()
