#!/usr/bin/env python
import logging
import argparse
import gensim

logging.basicConfig(format='%(asctime)s : %(levelname)s : %(message)s', level=logging.INFO)

class Corpus(object):
    def __init__(self, paths):
        self.paths = paths

    def __iter__(self):
        # train the model from a list of files
        for f in self.paths:
            for line in open(f):
                yield line.split()

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--input', nargs='+', required=True, help="training files")
    parser.add_argument('--output', help="path to the output model")
    parser.add_argument('--modelpath', help="path to a pretrained model")
    args = parser.parse_args()

    corpus = Corpus(args.input)

    if args.modelpath:
        model = gensim.models.Word2Vec.load(args.modelpath)
        model.build_vocab(sentences=corpus)
        model.train(sentences=corpus)
    else:
        model = gensim.models.Word2Vec(corpus, min_count=1, workers=4)

    try:
        model.save(args.output)
    except:
        fallback_path = "/tmp/model.word2vec"
        print("Failed to save model to %s, attempting to save to %s instead" % (args.output, fallback_path))
        model.save(fallback_path)


if __name__ == '__main__':
    main()
