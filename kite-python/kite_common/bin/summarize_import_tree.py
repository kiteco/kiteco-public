import argparse
import collections

from kite.ioutils.stream import loadjson

def main():
	parser = argparse.ArgumentParser()
	parser.add_argument("input")
	args = parser.parse_args()

	count_by_pkg = collections.defaultdict(int)

	with open(args.input) as f:
		for obj in loadjson(f):
			name = obj["canonical_name"]
			if name is not None:
				pos = name.find(".")
				if pos != -1:
					name = name[:pos]
				count_by_pkg[name] += 1

	for name, count in sorted(count_by_pkg.items(), key=lambda x: x[0]):
		print("%20s %d" % (name, count))


if __name__ == "__main__":
	main()
