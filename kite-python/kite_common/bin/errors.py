#!/usr/bin/env python

import os
import argparse

from kite.errorextraction import errors
from os import listdir, makedirs, path
from os.path import isfile, join

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("code_snippets_file_path")
    parser.add_argument("output_dir")
    args = parser.parse_args()

    extracted_errors = errors.extract_errors_from_file(args.code_snippets_file_path)
    output = errors.results_to_string(extracted_errors)

    output_dir = args.output_dir[:-1] if args.output_dir[-1] == '/' else args.output_dir
    
    with open(output_dir + "/output", "w+") as f:
        f.write(output)

if __name__ == "__main__":
    main()
