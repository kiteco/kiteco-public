#!/usr/bin/env python
import logging
import argparse
import gensim

logging.basicConfig(format='%(asctime)s : %(levelname)s : %(message)s', level=logging.INFO)

class Corpus(object):
	def __init__(self, path):
		self.path = path
	def __iter__(self):
		# we must re-open the file every time to allow two passes over the data
		for i, words in enumerate(map(str.split, open(self.path))):
			yield gensim.models.doc2vec.LabeledSentence(labels=['SNIPPET_%d'%i], words=words)


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument('input')
	parser.add_argument('output')
	args = parser.parse_args()

	corpus = Corpus(args.input)

	model = gensim.models.Doc2Vec(corpus, min_count=1, workers=4)
	try:
		model.save(args.output)
	except:
		fallback_path = "/tmp/model.doc2vec"
		print("Failed to save model to %s, attempting to save to %s instead" % (args.output, fallback_path))
		model.save(fallback_path)


if __name__ == '__main__':
	main()
