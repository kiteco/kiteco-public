#!/usr/bin/env python

import argparse

from kite.stacktraceextraction import stacktraces


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("code_snippets_file_path")
    parser.add_argument("output_dir")
    args = parser.parse_args()

    extracted_stacktraces = stacktraces.extract_stacktraces_from_file(args.code_snippets_file_path)
    output = stacktraces.results_to_string(extracted_stacktraces)

    output_dir = args.output_dir[:-1] if args.output_dir[-1] == '/' else args.output_dir
    
    with open(output_dir + "/output", "w+") as f:
        f.write(output)

if __name__ == "__main__":
    main()
