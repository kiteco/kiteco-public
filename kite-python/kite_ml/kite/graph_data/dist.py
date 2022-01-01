from typing import Dict, NamedTuple, NewType

from ..asserts.asserts import FieldValidator


class SymbolDistEntry(NamedTuple):
    symbol: str
    canonicalize: bool
    weight: float

    @classmethod
    def from_json(cls, d: dict) -> 'SymbolDistEntry':
        v = FieldValidator(cls, d)

        return SymbolDistEntry(
            symbol=v.get('symbol', str),
            canonicalize=v.get('canonicalize', bool),
            weight=v.get_float('weight'),
        )

    def to_json(self) -> dict:
        return {
            'symbol': self.symbol,
            'canonicalize': self.canonicalize,
            'weight': self.weight,
        }


SymbolDist = Dict[str, SymbolDistEntry]


def symbol_dist_from_json(d: dict) -> SymbolDist:
    return {k: SymbolDistEntry.from_json(v) for k, v in d.items()}


def symbol_dist_to_json(sd: SymbolDist) -> dict:
    return {k: v.to_json() for k, v in sd.items()}

