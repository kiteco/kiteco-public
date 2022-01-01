import sys
import argparse
import gensim


def main():
	modelpath = sys.argv[1]
	words = sys.argv[2:]

	positive = []
	negative = []
	for word in words:
		if word.startswith("-") and len(word) >= 2:
			negative.append(word[1:])
		else:
			positive.append(word)

	print("Positive:", positive)
	print("Negative:", negative)

	print('Loading model...')
	model = gensim.models.Word2Vec.load(modelpath)

	print('Performing query...')
	for result, dist in model.most_similar(positive=positive, negative=negative):
		print('%40s %.2f' % (result, dist))


if __name__ == '__main__':
	main()
