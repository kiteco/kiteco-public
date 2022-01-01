# This script extracts error templates from the golang source.
#
# Invocation example:
#
#     $ python2.7 kite-python/kite/canonicalization/extract.py ~/go/src/github.com/golang/go/ > golang-templates.txt

from __future__ import print_function

import os
import argparse
import subprocess



def find_string_literal(s):
    a = s.find('"')
    b = s.rfind('"')
    if a == -1 or b == -1:
        print('NO STRING LITERAL IN:', s)
        return None
    return s[a+1:b]

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('golang_source')
    args = parser.parse_args()

    try:
        parser_dir = os.path.join(args.golang_source, 'src/cmd/asm')
        parser_cmd = ['grep', 'p.errorf(".*"', '-o', '-R', '--no-filename', parser_dir]
        parser_lines = subprocess.check_output(parser_cmd).split('\n')

        checker_dir = os.path.join(args.golang_source, 'src/cmd/internal')    
        checker_cmd = ['grep', 'Yyerror(".*"', '-o', '-R', '--no-filename', checker_dir]
        checker_lines = subprocess.check_output(checker_cmd).split('\n')

        checker_cmd2 = ['grep', 'yyerrorl(.*, ".*"', '-o', '-R', '--no-filename', checker_dir]
        checker_lines2 = subprocess.check_output(checker_cmd2).split('\n')
    except subprocess.CalledProcessError as ex:
        print(ex.output)
        raise ex

    # Pull out string literals
    format_strings = []
    for line in parser_lines + checker_lines + checker_lines2:
        if ' ' in line:
            format_strings.append(find_string_literal(line))

    # Delete repetitions and sort
    format_strings = sorted(set(format_strings))
    
    # Print results
    for error_id, format_string in enumerate(format_strings):
        print('%d\t%s' % (error_id, format_string))


if __name__ == '__main__':
    main()
