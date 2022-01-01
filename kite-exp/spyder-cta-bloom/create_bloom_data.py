# this script takes output of ./spyder-data,
# adds all entries to a new bloomfilter
# and serializes the data of the bloomfilter into the output file
#
# usage: create_bloom_data.py <in> <out>

import sys
# this is our fork of pybloom to support PyQt bitset
# https://github.com/kiteco/python-bloomfilter
from pybloom_pyqt import BloomFilter

if len(sys.argv) != 3:
    print("usage: {} <input-file> <output-file>".format(sys.argv[0]))
    sys.exit(-1)
input_path = sys.argv[1]
output_path = sys.argv[2]

f = open(input_path, 'r')
lines = 0
for line in f.readlines():
    lines += 1

print('bloom filter size: {}'.format(lines))
f.seek(0)
b = BloomFilter(capacity=lines)
for line in f.readlines():
    b.add(line.strip('\n\r '))

f.close()

b.tofile(output_path)
print('successfully saved bloom filter data to {}'.format(output_path))

checks = {
    "json.load",
    "json.loads",
    "json.dumps",
    "plt.figure",
    "pd.read_csv",
}
for check in checks:
    print('selftest: {}? {}'.format(check, check in b))
