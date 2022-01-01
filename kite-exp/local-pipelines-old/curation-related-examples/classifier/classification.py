#!/usr/bin/env python
import sys
import json
import argparse
import pickle
import gensim

try:
    import queue as Q
except ImportError:
    import Queue as Q  # ver. < 3.0

from sklearn.externals import joblib

import kite.relatednessclassifier.svm as relatedness
from kite.emr import io

MAX_NUM_RELATED = 10
RELATED_THRESHOLD = 0.5

class RelatedExamples(object):
    def __init__(self, snippet_id, examples):
        self.snippet_id = snippet_id
        self.examples = examples

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--fout', help="output file name", default="related_examples.json")
    parser.add_argument('--svm', help="path to the svm classifier", required=True)
    parser.add_argument('--word2vec_model', help="path to the word2vec model", required=True)
    args = parser.parse_args()

    # open output file
    fout = open(args.fout, 'w')

    # set up the classifier and featurizer
    with open(args.svm, 'rb') as fin:
        clf = pickle.load(fin)

    word2vec_model = gensim.models.Word2Vec.load(args.word2vec_model)
    featurizer = relatedness.TitleRelatednessFeaturizer(word2vec_model)

    clf.featurizer = featurizer

    # read and process data
    prev_package = ''
    snippets = []

    for package, buf in io.read(sys.stdin):
        if package != prev_package and prev_package != '':
            relatedness_map = classify_snippet_pairs(clf, snippets)
            save_to_json(fout, relatedness_map)
            snippets = []

        snippet = json.loads(buf.decode('utf-8'))
        if snippet['Title'] != '' and snippet['Prelude'] != '':
            snippets.append(snippet)
        prev_package = package

    relatedness_map = classify_snippet_pairs(clf, snippets)
    save_to_json(fout, relatedness_map)

    # close output file
    fout.close()

def save_to_json(fout, relatedness_map):
    for s, related in relatedness_map.items():
        if related.empty():
            continue
        objs = []
        while not related.empty():
            objs.append(related.get()[1])
        # rank the code snippets from the most related to the least related
        objs = objs[::-1]
        re = RelatedExamples(s, objs)
        print(json.dumps(vars(re)), file=fout)


def classify_snippet_pairs(clf, snippets):
    """classify_snippet_pairs classifies all pairs of code snippets
    in a package.
    """
    relatedness_map = dict()

    # Initialize the map
    for s in snippets:
        relatedness_map[s['SnippetID']] = Q.PriorityQueue()

    for i, s_i in enumerate(snippets):
        tag_i = s_i['SnippetID']
        for j, s_j in enumerate(snippets):
            tag_j = s_j['SnippetID']
            if i < j:
                line = json.dumps({
                        'CodeA': s_i['Code'],
                        'CodeB': s_j['Code'],
                        'TitleA': s_i['Title'],
                        'TitleB': s_j['Title'],
                        'PreludeA': s_i['Prelude'],
                        'PreludeB': s_j['Prelude']})

                neg, pos = clf.predict_proba(line)[0]

                if pos >= RELATED_THRESHOLD:
                    relatedness_map[tag_i].put((pos, tag_j))
                    relatedness_map[tag_j].put((pos, tag_i))

                    if relatedness_map[tag_i].qsize() > MAX_NUM_RELATED:
                        relatedness_map[tag_i].get()

                    if relatedness_map[tag_j].qsize() > MAX_NUM_RELATED:
                        relatedness_map[tag_j].get()

    return relatedness_map

if __name__ == '__main__':
    main()
