# This script reads a file with error messages and produces a file with
# templates for error messages.
#
# Here's an example of invoking this script:
#
#    $ PYTHONPATH=$PYTHONPATH:~/kiteco/kite-python python bin/discover_error_templates.py ~/Downloads/golang_errors --out golang-templates.txt
#
# The input file should look like this:
#
#     ./server.go:28: non-integer array index "Name"
#     src/2rl/buildozer/buildozer.go:21: undefined: configpath
#     foo_test.go:35: Say hi
#     prog.go:12: invalid array bound n
#     ./wiki.go:97: undefined: addr
#     ./main.go:30: p1.save evaluated but not used
#     src/flow/graph_json.go:5: can't find import: "github.com/gyuho/goraph"
#     ./test.go:22: Print redeclared in this block
#     ./test.go:40: cannot use a (type *A) as type *B in function argument
#     ./wc.go:34: *x.PushBack(pair) evaluated but not used
#
# The output file will look like this:
#
#      0	%ca: %v
#      1	%ca: cannot create %s
#      2	%s
#      3	%s argument too large in make(%v)
#      4	%s discards result of %v
#      5	%s is not a type
#      6	%s is shadowed during return
#      7	%s overflows int64
#
# (that's a tab between the ids and templates)
#

import argparse
import itertools
import collections

import kite.canonicalization.discovery as discovery

def main():
	# Parse commandline arguments
	parser = argparse.ArgumentParser()
	parser.add_argument('input')
	parser.add_argument('--golang', action='store_true', default=False)
	parser.add_argument('--min_members', type=int, default=5)
	parser.add_argument('--out', type=str, required=False)
	parser.add_argument('--algorithm', type=str, default='flat_agglomerative')
	args = parser.parse_args()

	# Compile a list of error messages
	errorlines = []
	for line in open(args.input):
		line = line.strip()
		if args.golang:
			pos = line.find('.go:')
			if pos != -1:
				pos = line.find(' ', pos)
				if pos != -1:
					line = line[pos+1:]
		if 5 <= len(line) <= 200:
			errorlines.append(line)

	# Tokenize
	print('Tokenizing...')
	tokenvecs = list(map(discovery.tokenize, errorlines))

	# Categorize by first token ("IOError", "TypeError", etc)
	tokenvecs_by_head = collections.defaultdict(list)
	for tokenvec in tokenvecs:
		tokenvecs_by_head[tokenvec[0]].append(tokenvec)

	# Discover templates in each list separately
	members = []
	templates = []
	for head, vecs in tokenvecs_by_head.items():
		print('\n******************************\n%s\n******************************' % head)
		sub_templates, sub_members = discovery.discover_templates(
			vecs,
			min_members=args.min_members,
			algorithm=args.algorithm)
		templates.extend(sub_templates)
		members.extend(sub_members)

	# Write templates to file
	if args.out is not None:
		with open(args.out, 'w') as fd:
			for template_id, template in enumerate(templates):
				fd.write('%d\t%s\n' % (template_id, discovery.format_string_from_template(template)))

	# Report the templates
	print('\nAll templates:\n')
	print(' id count  error')
	for count, template_id, template in sorted(zip(list(map(len, members)), itertools.count(), templates)):
		if template is not None:
			print('%3d %5d  %s' % (template_id, count, discovery.format_string_from_template(template)))


if __name__ == '__main__':
	main()
