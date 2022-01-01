import collections
import argparse

# A simple utility to count the frequency of different python exception classes
# from a file containing lines of the form
#    PythonErrorType: message

def main():
	parser = argparse.ArgumentParser()
	parser.add_argument('input')
	args = parser.parse_args()

	counts = collections.defaultdict(int)
	for line in open(args.input):
		pos = line.find(':')
		if pos != -1:
			counts[line[:pos]] += 1

	for prefix, count in sorted(counts.items(), key=lambda v: v[1]):
		print('%10d: %s' % (count, prefix))

if __name__ == '__main__':
	main()
