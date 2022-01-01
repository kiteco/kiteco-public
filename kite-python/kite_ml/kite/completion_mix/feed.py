from typing import List, NamedTuple

from .raw_sample import RawSample, RawContextualFeatures, RawCompFeatures
from enum import Enum
import numpy as np


def encode_enum(enum_feature: Enum) -> List[float]:
    res = [0.0]*len(type(enum_feature))
    res[enum_feature.value] = 1.0
    return res


def encode_number(feature) -> List[float]:
    return [float(feature)]


def log_transform(feature_list) -> List[float]:
    log_feature = np.log(feature_list[0])
    if log_feature == -np.inf:
        return [0]
    return [log_feature]


class Feed(NamedTuple):
    context_features: List[List[float]]  # size: [batch size, feature depth]
    comp_features: List[List[float]]     # size: [total number of completions between all samples, feature depth]
    sample_ids: List[int]                # size: [total number of completions between all samples]
    labels: List[int]                    # size: [batch size]

    @classmethod
    def context_feature_vector(cls, context_features: RawContextualFeatures) -> List[float]:
        # TODO: Once the features and their corresponding representations are clear change feature to class

        # this 62-dimensional vector represents the 61 astTypes +1 for the number of variables.
        vectorized_context = [0.0]*62
        vectorized_context[context_features.parent_type-1] = 1.0
        log_num_vars = np.log(context_features.num_vars)
        if log_num_vars == -np.inf:
            log_num_vars = 0.0
        vectorized_context[61] = log_num_vars

        # vectorized_context[61] = float(context_features.num_vars)

        return vectorized_context

    @classmethod
    def comp_feature_vector(cls, comp_features: RawCompFeatures) -> List[float]:
        # TODO: Once the features and their corresponding representations are clear change feature to class
        # this 9-dimensional vector represents 3 score +4 model type +1 completion_length + 1 num_args
        vectorized_comp = []
        for attr in ['popularity_score', 'softmax_score', 'attr_score', 'model', 'completion_length', 'num_args']:
            val = getattr(comp_features, attr)
            if type(val) == int or type(val) == float:
                v = encode_number(val)
                if attr == 'popularity_score' or attr == 'softmax_score' or attr == 'attr_score':
                    v = log_transform(v)
                vectorized_comp += v
            else:
                vectorized_comp += encode_enum(val)
        return vectorized_comp

    @staticmethod
    def context_feature_depth() -> int:
        return 62

    @staticmethod
    def comp_feature_depth() -> int:
        return 9

    @classmethod
    def from_samples(cls, batch_samples: List[RawSample]):
        def context_vect(s: RawSample) -> List[float]:
            v = Feed.context_feature_vector(s.context_features)
            assert len(v) == cls.context_feature_depth(), \
                "mismatch between size of context features ({}) and declared depth ({})".format(
                    len(v), cls.context_feature_depth())
            return v

        def comp_vect(comp: RawCompFeatures) -> List[float]:
            v = cls.comp_feature_vector(comp)
            assert len(v) == cls.comp_feature_depth(), \
                "mismatch between size of per-completion features ({}) and declared depth ({})".format(
                    len(v), cls.comp_feature_depth())
            return v

        context_features: List[List[float]] = []
        comp_features: List[List[float]] = []
        sample_ids: List[int] = []
        labels: List[int] = []

        offset = 0
        for i, sample in enumerate(batch_samples):
            context_features.append(context_vect(sample))
            labels.append(sample.label + offset)
            offset += len(sample.comp_features)
            for c in sample.comp_features:
                comp_features.append(comp_vect(c))
                sample_ids.append(i)

        return Feed(
            context_features=context_features,
            comp_features=comp_features,
            sample_ids=sample_ids,
            labels=labels,
        )
