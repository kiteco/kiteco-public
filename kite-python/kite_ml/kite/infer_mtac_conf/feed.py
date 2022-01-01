from typing import List, NamedTuple

import math

from .raw_sample import RawSample, RawContextualFeatures, RawCompFeatures
from enum import Enum


def encode_enum(enum_feature: Enum) -> List[float]:
    res = [0.0]*len(type(enum_feature))
    res[enum_feature.value] = 1.0
    return res


def encode_number(feature) -> List[float]:
    return [float(feature)]


class Feed(NamedTuple):
    contextual_features: List[List[float]]  # size: [batch size, feature depth]
    comp_features: List[List[float]]     # size: [total number of completions between all samples, feature depth]
    sample_ids: List[int]                # size: [total number of completions between all samples]
    labels: List[int]                    # size: [total number of completions before all samples]

    @classmethod
    def contextual_feature_vector(cls, contextual_features: RawContextualFeatures) -> List[float]:
        return [1.0,  # intercept
                math.log(float(contextual_features.num_vars)) if contextual_features.num_vars > 0 else 0.0,
                ]

    @staticmethod
    def contextual_feature_depth() -> int:
        return 2

    @classmethod
    def comp_feature_vector(cls, comp_features: RawCompFeatures) -> List[float]:
        comp_vec = [float(comp_features.score),
                    float(comp_features.return_type_percent_matched),
                    float(comp_features.is_iterable),
                    float(comp_features.none_ratio),
                    float(comp_features.comp_types_empty)]
        scope_vec = encode_enum(comp_features.scenario)
        comp_vec = comp_vec+scope_vec
        return comp_vec

    @staticmethod
    def comp_feature_depth() -> int:
        return 9

    @classmethod
    def from_samples(cls, batch_samples: List[RawSample]):
        def contextual_vect(s: RawSample) -> List[float]:
            v = Feed.contextual_feature_vector(s.features.contextual)
            assert len(v) == cls.contextual_feature_depth(), \
                "mismatch between size of context features ({}) and declared depth ({})".format(
                    len(v), cls.contextual_feature_depth())
            return v

        def comp_vect(comp: RawCompFeatures) -> List[float]:
            v = cls.comp_feature_vector(comp)
            assert len(v) == cls.comp_feature_depth(), \
                "mismatch between size of per-completion features ({}) and declared depth ({})".format(
                    len(v), cls.comp_feature_depth())
            return v

        contextual_features: List[List[float]] = []
        comp_features: List[List[float]] = []
        sample_ids: List[int] = []
        labels: List[int] = []

        for i, sample in enumerate(batch_samples):
            contextual_features.append(contextual_vect(sample))

            for j, c in enumerate(sample.features.comp):
                comp_features.append(comp_vect(c))
                sample_ids.append(i)
                labels.append(1 if j == sample.label else 0)

        return Feed(
            contextual_features=contextual_features,
            comp_features=comp_features,
            sample_ids=sample_ids,
            labels=labels,
        )
