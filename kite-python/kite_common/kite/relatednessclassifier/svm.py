import re
import json

from numpy import ones 

import kite.classification.svm as svm
import kite.classification.utils as utils


class TitleRelatednessFeaturizer(svm.SVMFeaturizer):

    """TitleRelatednessFeaturizer converts a pair of curated code snippets
    into a feature vector for determining whether two titles are related.


    It requires a pretrained word2vec model for generating features. It
    generates 3 types of features, which are described as follows.

    1) The cosine similarity of the functions in the two code snippets
    2) The cosine similarity of the verb phrases used in the two snippets
    3) The cosine similarity of the specifications used in the snippets
    """

    def __init__(self, model):
        self.model = model
        # Regular expressions for parsing import methods, verb phrases and sepcs
        # from titles.
        self.funccall_pattern = re.compile('[\.]*([a-zA-Z_\.]+)\(')
        self.prelude_pattern = re.compile(
            '(from )*(?P<package>.*)import (?P<methods>.+)')
        self.import_as_pattern = re.compile(
            '(from )*(?P<package>.*)import (?P<methods>.+) as (?P<alias>.+)')
        self.title_pattern = re.compile('(?P<vp>[^\[\]]+)(?P<specs>.*)')
        self.specs_pattern = re.compile('\[([^\[\]]+)\]')

    def features(self, line):
        self.data = json.loads(line)

        title1 = normalize_title(self.data['TitleA'])
        title2 = normalize_title(self.data['TitleB'])

        code1 = self.data['CodeA']
        code2 = self.data['CodeB']

        prelude1 = self.data['PreludeA']
        prelude2 = self.data['PreludeB']

        methods1 = self.get_methods(prelude1, code1)
        methods2 = self.get_methods(prelude2, code2)

        vp1, specs1 = self.parse_title(title1)
        vp2, specs2 = self.parse_title(title2)

        feat_vec = self.method_distance(methods1, methods2)
        feat_vec.append(self.vp_distance(vp1, vp2))
        feat_vec.extend(self.specs_distance(specs1, specs2))

        return feat_vec

    def parse_title(self, title):
        """Parse verb phrases and specifications out of a title"""
        m = self.title_pattern.match(title)
        vp = m.group('vp')
        specs = self.specs_pattern.findall(m.group('specs'))
        return vp, specs

    def get_methods(self, prelude, code):
        """Get the methods used in a code snippet"""
        # Get import methods and packages
        imported_packages = []
        imported_methods = []
        for line in prelude.split('\n'):
            m = self.import_as_pattern.search(line)
            if m is None:
                m = self.prelude_pattern.search(line)
            if m is not None:
                package = m.group('package')
                # import re, math
                if package == "":
                    packages = m.group('methods').split(",")
                    for package in packages:
                        imported_packages.append(package)
                        imported_methods.append("")
                else:
                    # from re import compile, group
                    for method in m.group('methods').split(","):
                        imported_methods.append(method.strip())
                        imported_packages.append(package)

        # Get methods that are used in the code
        func_calls = []
        for line in code.split('\n'):
            methods = self.funccall_pattern.findall(line)
            for f in methods:
                for i, m in enumerate(imported_methods):
                    p = imported_packages[i]
                    if m in f or p in f:
                        func_calls.append(
                            ' '.join([p, f.split('.')[-1]]).strip())
        if len(func_calls) > 0:
            return func_calls
        else:
            return [' '.join([imported_packages[i], m]).strip()
                    for i, m in enumerate(imported_methods)]

    def specs_distance(self, specs1, specs2):
        """Compute a semantic vector for each spec and compute the distance
        between each pair of specs.
        """
        scores = []
        for s1 in specs1:
            vec1 = self.phrase2vec(s1.split())
            for s2 in specs2:
                vec2 = self.phrase2vec(s2.split())
                scores.append(utils.cosine_similarity(vec1, vec2))
        try:
            return [sum(scores) / len(scores), max(scores), min(scores)]
        except:
            return [0.5, 0.5, 0.5]


    def vp_distance(self, vp1, vp2):
        """Compute a semantic vector for each verb phrase and compute the distance
        between two verb phrases.
        """
        vec1 = self.phrase2vec(vp1.split())
        vec2 = self.phrase2vec(vp2.split())
        return utils.cosine_similarity(vec1, vec2)

    def method_distance(self, methods1, methods2):
        """Compute the distance between two arrays of methods."""
        scores = []

        # normalize methods
        normalized_methods1 = []
        for m in methods1:
            normalized_methods1.append(normalize_package_name(m))

        normalized_methods2 = []
        for m in methods2:
            normalized_methods2.append(normalize_package_name(m))

        for m1 in normalized_methods1:
            vec1 = self.phrase2vec(m1)
            for m2 in normalized_methods2: 
                vec2 = self.phrase2vec(m2)
                scores.append(utils.cosine_similarity(vec1, vec2))
        try:
            return [sum(scores) / len(scores), max(scores), min(scores)]
        except:
            return [0.0, 0.0, 0.0] 

    def phrase2vec(self, words):
        """Convert an array of words to a vector."""
        try:
            return sum([self.model[t] for t in words]) / len(words)
        except:
            return ones(self.model.layer1_size)


def normalize_title(s):
    s = s.lower()
    s = s.replace('`s', '')
    s = s.replace('`', '')
    return s


def normalize_package_name(s):
    """normalize_package_name formats the package name by following the style
    of the tokenizer of go.

    For example, "numpy.linalg" is mapped to ["numpy", ".", "linalg"]
    """
    try:
        p, m = s.split()
    except:
        # no package name exists, so just return the method
        return s.split() 
    tokens = p.split('.')
    i = 1
    while i < len(tokens):
        tokens.insert(i, ".")
        i += 2
    return tokens + [m]
