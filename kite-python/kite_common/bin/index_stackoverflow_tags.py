import collections
import argparse
import sqlite3


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument('database')
	args = parser.parse_args()

	conn = sqlite3.connect(args.database)
	c = conn.cursor()

	c.execute('DROP TABLE IF EXISTS tags')
	c.execute('CREATE TABLE tags (tag TEXT, post_id INTEGER)')

	tags = []
	counts = collections.defaultdict(int)

	for i, (post_id, tagstr) in enumerate(c.execute('SELECT id, tags FROM posts')):
		if i+1 % 100000 == 0:
			print( 'Processing post %d' % (i+1))
		if tagstr is not None:
			for tag in tagstr.split():
				counts[tag] += 1
				tags.append((tag, post_id))

	print('Inserting %d tags' % len(tags))
	c.executemany('INSERT INTO tags VALUES (?, ?)', tags)

	conn.commit()
	c.close()

	# Print a summary
	for tag, count in sorted(counts.items(), key=lambda v: v[1]):
		print('%10d %s' % (count, tag))

	print('To get anywhere near remove performance you should now do:')
	print('sqlite3> create index code_blocks_post_id on code_blocks(post_id);')
	print('sqlite3> create index tags_tag on tags(tag);')

if __name__ == '__main__':
	main()
