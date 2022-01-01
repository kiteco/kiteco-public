"""
Given:
  an error ontology as constructed by discover_error_templates.py
  and an sqlite database containing stackoverflow data as constructed by stackoverflow_to_sqlite.py
Construct:
  an index that maps canonicalized error IDs to stackoverflow post IDs

The index is encoded in json in the format
{
	"source_stackoverflow_database": "/path/to/posts.sqlite",
	"source_ontology": "/path/to/error-templates.txt",
	"items": [
		{
			"error_id": 123,
			"post_ids": [1, 2, 3, ...]
		},
		...
	]
}
"""

import re
import json
import sqlite3
import argparse
import collections

import kite.canonicalization.ontology as ontology


GOLANG_PATTERN = re.compile(r'(([^:]*\.go)\:(\d+)([^\d][^ ]*)?\:\ )(.*)')

def extract_error_golang(s):
	r = GOLANG_PATTERN.match(s)
	if r is None:
		return None
	else:
		return r.group(5)


def extract_error_python(s):
	if 'Error:' in s:
		return s
	else:
		return None


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument('stackoverflow_db')
	parser.add_argument('ontology')
	parser.add_argument('output')
	parser.add_argument('--max', type=int, default=100, required=False)
	parser.add_argument('--language', type=str, choices=['python', 'golang'], required=True)
	args = parser.parse_args()

	# Load ontology
	recog = ontology.Ontology(list(open(args.ontology)))

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

	num_lines = 0
	num_error_lines = 0
	num_canonicalized = 0
	rows_processed = 0
	while True:
		batch = c.fetchmany(1000)
		if len(batch) == 0:
			break

		for content, post_id in batch:
			post_id = int(post_id)
			for line in content.split('\n'):
				num_lines += 1
				errmsg = extract_error(line)
				if errmsg is not None:
					print('Found error:', line)
					num_error_lines += 1
					msg = recog.canonicalize(errmsg)
					if msg is not None:
						print('  Was canonicalized to:', msg.pattern.format_string.strip())
						num_canonicalized += 1
						post_ids_by_pattern_index[msg.pattern.index].append(post_id)

		rows_processed += len(batch)
		print('%d rows processed' % rows_processed)

	# Create the index structure
	items = []
	for error_id, post_ids in post_ids_by_pattern_index.items():
		if len(post_ids) <= args.max:
			items.append(dict(error_id=error_id, post_ids=post_ids))

	results = {
		"source_stackoverflow_database": args.stackoverflow_db,
		"source_ontology": args.ontology,
		"items": items,
	}

	with open(args.output, 'w') as f:
		json.dump(results, f)

	print("\nQuery was:\n%s\n" % sql)
	print("Query returned %d rows containing %d lines" % (rows_processed, num_lines))
	print("Of which %d matched the error pattern, of which %d were canonicalized" %
		(num_error_lines, num_canonicalized))
	print("%d patterns were found in the dataset (of %d total)" %
		(len(post_ids_by_pattern_index), len(recog.patterns)))


if __name__ == '__main__':
	import cProfile, pstats
	pr = cProfile.Profile()
	pr.enable()
	main()
	pr.disable()
	print('\n')
	pstats.Stats(pr).sort_stats('tottime').print_stats(12)
