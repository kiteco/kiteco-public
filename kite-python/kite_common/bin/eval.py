#!/usr/bin/env python

import os
import sys
import argparse
import getpass
import tempfile

from kite.errorextraction import errors
from kite.stacktraceextraction import stacktraces


ERRORS = "errors"
STACKTRACES = "stacktraces"

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("errors_or_stacktraces")
    parser.add_argument("ground_truth_file_path")
    parser.add_argument("output_dir")
    args = parser.parse_args()

    if args.errors_or_stacktraces == ERRORS:
        eval_results = errors.evaluate_ground_truth(args.ground_truth_file_path)
        output = errors.results_to_string(eval_results)
    elif args.errors_or_stacktraces == STACKTRACES:
        eval_results = stacktraces.evaluate_ground_truth(args.ground_truth_file_path)
        output = stacktraces.results_to_string(eval_results)
    else:
        print("Please specify either errors or stacktraces before the file path")
        return

    output_dir = args.output_dir[:-1] if args.output_dir[-1] == '/' else args.output_dir

    with open(output_dir + "/output", 'a') as f:
        f.write(output)

if __name__ == "__main__":
    main()
