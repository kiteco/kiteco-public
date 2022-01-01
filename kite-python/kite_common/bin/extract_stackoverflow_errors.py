"""
Given:
  an sqlite database containing stackoverflow data as constructed by stackoverflow_to_sqlite.py
Construct:
  a text file with one error message per line
"""

import re
import json
import sqlite3
import argparse
import collections


GOLANG_PATTERN = re.compile(r'(([^:]*\.go)\:(\d+)([^\d][^ ]*)?\:\ )(.*)')

def extract_error_golang(s):
	r = GOLANG_PATTERN.match(s)
	if r is None:
		return None
	else:
		return s


def extract_error_python(s):
	if 'Error:' in s:
		return s
	else:
		return None


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument('stackoverflow_db')
	parser.add_argument('output')
	parser.add_argument('--language', type=str, choices=['python', 'golang'], required=True)
	args = parser.parse_args()

	# Initialize inverted index
	post_ids_by_pattern_index = collections.defaultdict(list)

	# Connect to DB
	conn = sqlite3.connect(args.stackoverflow_db)
	c = conn.cursor()

	sql = """SELECT content, code_blocks.post_id 
	FROM code_blocks JOIN tags
	ON code_blocks.post_id == tags.post_id
	WHERE tag == ?"""

	if args.language == "python":
		extract_error = extract_error_python
		language_tag = "python"
	elif args.language == "golang":
		extract_error = extract_error_golang
		language_tag = "go"
	else:
		print("Uncorecognized language: %s" % args.language)
		return

	c.execute(sql, (language_tag,))

	num_rows = 0
	num_lines = 0
	num_errors = 0

	with open(args.output, 'w') as f:
		while True:
			batch = c.fetchmany(1000)
			if len(batch) == 0:
				break

			num_rows += len(batch)

			for content, post_id in batch:
				post_id = int(post_id)
				for line in content.split('\n'):
					num_lines += 1
					errmsg = extract_error(line)
					if errmsg is not None:
						f.write(errmsg.strip() + "\n")
						num_errors += 1

	print("\nQuery was:\n%s\n" % sql)
	print("Query returned %d rows containing %d lines, of which %d were errors" % 
		(num_rows, num_lines, num_errors))
	print("Wrote errors to %s" % args.output)


if __name__ == '__main__':
	import cProfile, pstats
	pr = cProfile.Profile()
	pr.enable()
	main()
	pr.disable()
	print('\n')
	pstats.Stats(pr).sort_stats('tottime').print_stats(12)
