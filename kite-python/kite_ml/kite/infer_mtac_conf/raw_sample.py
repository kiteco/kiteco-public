from enum import Enum
from typing import List, NamedTuple

from ..asserts.asserts import FieldValidator


class Scenario(Enum):
    IN_CALL = 0
    IN_WHILE = 1
    IN_IF = 2
    IN_FOR = 3


class RawContextualFeatures(NamedTuple):
    num_vars:    int

    @classmethod
    def from_json(cls, d: dict) -> 'RawContextualFeatures':
        v = FieldValidator(cls, d)

        return RawContextualFeatures(
            num_vars=v.get('num_vars', int),
        )


class RawCompFeatures(NamedTuple):
    score: float
    return_type_percent_matched: float
    is_iterable: int
    none_ratio: float
    comp_types_empty: int
    scenario: Scenario

    @classmethod
    def from_json(cls, d: dict) -> 'RawCompFeatures':
        v = FieldValidator(cls, d)

        return RawCompFeatures(
            score=v.get_float('score'),
            return_type_percent_matched=v.get_float('return_type_percent_matched'),
            is_iterable=v.get('is_iterable', int),
            none_ratio=v.get_float('none_ratio'),
            comp_types_empty=v.get('comp_types_empty', int),
            scenario=v.get_enum('scenario', Scenario)
        )


class RawFeatures(NamedTuple):
    contextual: RawContextualFeatures
    comp: List[RawCompFeatures]

    @classmethod
    def from_json(cls, d: dict) -> 'RawFeatures':
        v = FieldValidator(cls, d)

        return RawFeatures(
            contextual=v.get('contextual', dict, build=RawContextualFeatures.from_json),
            comp=v.get_list('comp', dict, build_elem=RawCompFeatures.from_json, min_len=1),
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
    features: RawFeatures

    label: int
    meta: RawSampleMeta

    @classmethod
    def from_json(cls, d: dict) -> 'RawSample':
        v = FieldValidator(cls, d)

        features: RawFeatures = v.get('features', dict, build=RawFeatures.from_json)
        label = v.get('label', int)

        assert -1 <= label < len(features.comp), \
            "label ({}) out of range of comp feature length ({})".format(label, len(features.comp))

        return RawSample(
            features=features,
            label=label,
            meta=v.get('meta', dict, build=RawSampleMeta.from_json),
        )
