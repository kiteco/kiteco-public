from typing import List, NamedTuple, Dict

from ..asserts.asserts import Assert, FieldValidator

from .graph_feed import GraphFeedConfig
from ..infer_expr.feed import RawTrainSample as RawExprTrainSample

from ..infer_expr.request import Request as ExprRequest

from ..infer_production.index import Index as ProductionIndex

DEFAULT_RANDOM_SEED = 243


class RawTrainData(NamedTuple):
    expr: RawExprTrainSample

    @classmethod
    def from_json(cls, d: dict) -> 'RawTrainData':
        v = FieldValidator(cls, d)
        return RawTrainData(
            expr=v.get('expr', dict, build=RawExprTrainSample.from_json),
        )


class RawSample(NamedTuple):
    data: RawTrainData

    @classmethod
    def from_json(cls, d: dict) -> 'RawSample':
        v = FieldValidator(cls, d)

        return RawSample(
            data=v.get('data', dict, build=RawTrainData.from_json),
        )


class RawSessionResponse(NamedTuple):
    session: int
    samples: List[RawSample]

    @classmethod
    def from_json(cls, d: dict) -> 'RawSessionResponse':
        v = FieldValidator(cls, d)

        return RawSessionResponse(
            session=v.get('session', int, Assert.greater_than_or_equal(1)),
            samples=v.get_list('samples', dict, min_len=1,
                               build_elem=RawSample.from_json),
        )


class Partition(NamedTuple):
    low: float = 0.0
    high: float = 1.0

    def to_json(self) -> dict:
        return {
            'low': self.low,
            'high': self.high,
        }


class RequestInit(NamedTuple):
    config: GraphFeedConfig
    name_subtoken_index: Dict[str, int]
    type_subtoken_index: Dict[str, int]
    production_index: ProductionIndex
    expr: ExprRequest
    random_seed: int = DEFAULT_RANDOM_SEED
    partition: Partition = Partition()
    num_batches: int = 10
    max_hops: int = 0

    def to_json(self) -> dict:
        return {
            'session': 0,
            'config': self.config.to_json(),
            'name_subtoken_index': self.name_subtoken_index,
            'type_subtoken_index': self.type_subtoken_index,
            'production_index': self.production_index.to_json(),
            'random_seed': self.random_seed,
            'partition': self.partition.to_json(),
            'num_batches': self.num_batches,
            'max_hops': self.max_hops,
            'expr': self.expr.to_json(),
        }


class Request(NamedTuple):
    session: int

    def to_json(self) -> dict:
        return {
            'session': self.session,
        }
