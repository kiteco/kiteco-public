from typing import List, NamedTuple

from ..asserts.asserts import FieldValidator


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
    num_args: int
    pattern_freq: float
    type_match_score: float
    types_violated: float
    pattern_match: float
    effective_args: float
    subtok_match_score: float
    subtoks_violated: float
    placeholder_count: float
    placeholder_scope_ratio: float

    @classmethod
    def from_json(cls, d: dict) -> 'RawCompFeatures':
        v = FieldValidator(cls, d)

        return RawCompFeatures(
            score=v.get_float('score'),
            num_args=v.get('num_args', int),
            pattern_freq=v.get_float('pattern_freq'),
            type_match_score=v.get_float('type_match_score'),
            types_violated=v.get_float('types_violated'),
            pattern_match=v.get_float('pattern_match'),
            effective_args=v.get_float('effective_args'),
            subtok_match_score=v.get_float('subtok_match_score'),
            subtoks_violated=v.get_float('subtoks_violated'),
            placeholder_count=v.get_float('placeholder_count'),
            placeholder_scope_ratio=v.get_float('placeholder_scope_ratio')
        )


class RawFeatures(NamedTuple):
    contextual: RawContextualFeatures
    comp: RawCompFeatures

    @classmethod
    def from_json(cls, d: dict) -> 'RawFeatures':
        v = FieldValidator(cls, d)

        return RawFeatures(
            contextual=v.get('contextual', dict, build=RawContextualFeatures.from_json),
            comp=v.get('comp', dict, build=RawCompFeatures.from_json),
        )


class RawSampleMeta(NamedTuple):
    hash: str
    cursor: int
    comp_identifier: str

    @classmethod
    def from_json(cls, d: dict) -> 'RawSampleMeta':
        v = FieldValidator(cls, d)

        return RawSampleMeta(
            hash=v.get('hash', str),
            cursor=v.get('cursor', int),
            comp_identifier=v.get('comp_identifier', str),
        )


class RawSample(NamedTuple):
    features: RawFeatures

    label: bool
    meta: RawSampleMeta

    @classmethod
    def from_json(cls, d: dict) -> 'RawSample':
        v = FieldValidator(cls, d)
        features: RawFeatures = v.get('features', dict, build=RawFeatures.from_json)
        label = v.get('label', bool)

        return RawSample(
            features=features,
            label=label,
            meta=v.get('meta', dict, build=RawSampleMeta.from_json),
        )
