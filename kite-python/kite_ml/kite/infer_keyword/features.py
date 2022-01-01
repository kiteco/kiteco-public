from typing import Callable, List

import tensorflow as tf

from .constants import Constants
from .config import Config
from .raw import RawRecord


class Feature(object):
    def __init__(self, in_size: int, out_size: int):
        self.in_size = in_size
        self.out_size = out_size

    # feature_list extracts the relevant information from the raw record and returns a list containing the features
    # which can be fed into Tensorflow.
    def feature_list(self, record: RawRecord) -> List[int]:
        raise NotImplementedError("need to implement feature_list")

    # encode_op should take a slice of the source tensor, of size (batch_size, in_size), and outputs a tensor of size
    # (batch_size, out_size).
    def encode_op(self, source_slice: tf.Tensor) -> tf.Tensor:
        raise NotImplementedError("need to implement op")


class Categorical(Feature):
    def __init__(self, extractor: Callable[[RawRecord], List[int]], cardinality: int, count: int=1):
        super().__init__(count, count * cardinality)
        self.extractor = extractor
        self.count = count
        self.cardinality = cardinality

    def feature_list(self, record: RawRecord) -> List[int]:
        extracted = self.extractor(record)
        assert len(extracted) == self.count
        return extracted

    def encode_op(self, source_slice: tf.Tensor) -> tf.Tensor:
        one_hot = tf.one_hot(source_slice, self.cardinality)
        # tf.one_hot appends an extra dimension (size = cardinality) to the input tensor, so we'd end up with
        # a tensor of size (batch_size, count, cardinality). We apply tf.reshape in order to convert it back
        # to a 2D tensor.
        return tf.reshape(one_hot, (-1, self.out_size))


class BagOfItems(Feature):
    """
    BagOfItems groups a serie of categorical items in one `bag`
    The operator to do the reduction is reduce_sum so it's a bag of count (and not a bag of booleans)"""
    def __init__(self, extractor: Callable[[RawRecord], List[int]], cardinality: int, count: int=1, ngrams: int = 5):
        super().__init__(ngrams, cardinality)
        self.ngrams = ngrams
        self.extractor = extractor
        self.count = count
        self.cardinality = cardinality

    def feature_list(self, record: RawRecord) -> List[int]:
        extracted = self.extractor(record)
        # assert len(extracted) <= self.ngrams-1
        return extracted

    def encode_op(self, source_slice: tf.Tensor) -> tf.Tensor:
        one_hot = tf.one_hot(source_slice, self.cardinality)
        s = tf.reduce_sum(one_hot, axis=1)
        return s


class Integral(Feature):
    def __init__(self, extractor: Callable[[RawRecord], List[int]], count: int=1):
        super().__init__(count, count)
        self.extractor = extractor
        self.count = count

    def feature_list(self, record: RawRecord) -> List[int]:
        extracted = self.extractor(record)
        assert len(extracted) == self.count
        return extracted

    def encode_op(self, source_slice: tf.Tensor) -> tf.Tensor:
        return tf.cast(source_slice, tf.float32)


# FeatureEncoder performs two transformations:
# - feature_list: from a RawRecord to a list of integer features, which can be fed to Tensorflow.
# - encode_op: from a feature tensor to a transformed tensor, which can be fed to a classifier. This is implemented
#   as a composition of Tensorflow ops.
# The reason that we have two transformations is that the latter one can be part of the exported Tensorflow graph,
# and therefore can be used during inference and doesn't need to be re-implemented in Golang. However, the first
# transformation (feature_list) does need to be reimplemented for inference, since the exported model will expect
# a feature list with the same format.
class FeatureEncoder(object):
    def __init__(self, features: List[Feature]):
        self.features = features

    # in_size returns the size of the input feature list which is fed to Tensorflow.
    def in_size(self) -> int:
        return sum([f.in_size for f in self.features])

    # out_size returns the size of the transformed output tensor, encoded via encode_op, which can be fed directly
    # into a classifier.
    def out_size(self) -> int:
        return sum([f.out_size for f in self.features])

    # feature_list returns a list of integers which can be fed into the Tensorflow graph and subsequently encoded
    # via encode_op.
    def feature_list(self, record: RawRecord) -> List[int]:
        components = [f.feature_list(record) for f in self.features]
        feature_list = sum(components, [])
        assert len(feature_list) == self.in_size()
        return feature_list

    # encode_op transforms the source tensor of size (batch_size, in_size()) into a tensor of size
    # (batch_size, out_size()) that can be used directly with a classifier.
    def encode_op(self, source: tf.Tensor, name: str="encoder") -> tf.Tensor:
        i = 0
        tensors: List[tf.Tensor] = []
        for feature in self.features:
            source_slice = source[:, i:(i + feature.in_size)]
            tensors.append(feature.encode_op(source_slice))
            i += feature.in_size

        out_tensor: tf.Tensor = tf.concat(tensors, axis=1, name=name)
        assert int(out_tensor.shape[1]) == self.out_size()
        return out_tensor

    def get_features_str(self):
        return "["+" , ".join([f.describe() for f in self.features])+"]"

# KeywordModelEncoder is a FeatureEncoder specific to the keyword model.
class KeywordModelEncoder(FeatureEncoder):
    def __init__(self, config: Config):
        super().__init__(self._get_features(config))

    @staticmethod
    def _get_features(config: Config) -> List[Feature]:
        class Nodes(Categorical):
            def __init__(self):
                super().__init__(self.extractor, cardinality=Constants.N_NODES, count=2)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                return [rec.features.last_sibling, rec.features.parent_node]

            def describe(self):
                return "Last sibling and parent node info"

        class FirstToken(Categorical):
            def __init__(self):
                super().__init__(self.extractor, cardinality=Constants.N_TOKENS, count=1)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                return [rec.features.first_token]

            def describe(self):
                return "First token of current stmt"

        class RelIndent(Categorical):
            def __init__(self):
                super().__init__(self.extractor, cardinality=Constants.N_REL_INDENT, count=1)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                return [rec.features.rel_indent]

            def describe(self):
                return "Indentation level"

        class FirthChar(Categorical):
            def __init__(self):
                super().__init__(self.extractor, cardinality=Constants.N_PREFIXES, count=1)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                return [rec.features.first_char]

            def describe(self):
                return "First character (lower case letter only)"

        class PrevKeywords(Integral):
            def __init__(self):
                super().__init__(self.extractor, count=Constants.N_KEYWORDS)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                # The record might have more than N tokens, so take the last N
                return rec.features.previous_keywords

            def describe(self):
                return "Which keyword are already present in doc".format(config.lookback)


        class PrevTokens(Categorical):
            def __init__(self):
                super().__init__(self.extractor, cardinality=Constants.N_TOKENS, count=config.lookback)

            @staticmethod
            def extractor(rec: RawRecord) -> List[int]:
                # The record might have more than N tokens, so take the last N
                return rec.features.prev_tokens[-config.lookback:]

            def describe(self):
                return "Previous {} tokens (per category)".format(config.lookback)


        return [Nodes(), FirstToken(), RelIndent(), PrevTokens(), FirthChar(), PrevKeywords()]
