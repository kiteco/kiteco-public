from enum import Enum
from typing import List, NamedTuple

from ..asserts.asserts import FieldValidator


class CompType(Enum):
    SINGLE_POPULARITY = 0
    SINGLE_GGNN_ATTRIBUTE = 1
    MULTI_GGNN_ATTRIBUTE = 2
    CALL_GGNN = 3


class RawContextualFeatures(NamedTuple):
    parent_type: int
    num_vars:    int

    @classmethod
    def from_json(cls, d: dict) -> 'RawContextualFeatures':
        v = FieldValidator(cls, d)

        return RawContextualFeatures(
            parent_type=v.get('parent_type', int),
            num_vars=v.get('num_vars', int),
        )


class RawCompFeatures(NamedTuple):
    popularity_score: float
    softmax_score: float
    model: CompType
    completion_length: int
    num_args: int
    attr_score: float

    @classmethod
    def from_json(cls, d: dict) -> 'RawCompFeatures':
        v = FieldValidator(cls, d)

        return RawCompFeatures(
            popularity_score=v.get_float('popularity_score'),
            softmax_score=v.get_float('softmax_score'),
            model=v.get_enum('model', CompType),
            completion_length=v.get('completion_length', int),
            num_args=v.get('num_args', int),
            attr_score=v.get_float('attr_score')
        )


class RawSampleMeta(NamedTuple):
    hash: str
    cursor: int
    comp_identifiers: List[str]

    @classmethod
    def from_json(cls, d: dict) -> 'RawSampleMeta':
        v = FieldValidator(cls, d)

        return RawSampleMeta(
            hash=v.get('hash', str),
            cursor=v.get('cursor', int),
            comp_identifiers=v.get_list('comp_identifiers', str),
        )


class RawSample(NamedTuple):
    context_features: RawContextualFeatures
    comp_features: List[RawCompFeatures]
    label: int
    meta: RawSampleMeta

    @classmethod
    def from_json(cls, d: dict) -> 'RawSample':
        v = FieldValidator(cls, d)

        comp_features = v.get_list('comp_features', dict, build_elem=RawCompFeatures.from_json, min_len=1)
        label = v.get('label', int)

        assert 0 <= label < len(comp_features), \
            "label ({}) out of range of comp feature length ({})".format(label, len(comp_features))

        return RawSample(
            context_features=v.get('context_features', dict, build=RawContextualFeatures.from_json),
            comp_features=comp_features,
            label=label,
            meta=v.get('meta', dict, build=RawSampleMeta.from_json),
        )
