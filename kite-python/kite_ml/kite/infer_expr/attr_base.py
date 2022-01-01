from typing import NamedTuple

from ..asserts.asserts import FieldValidator

from ..graph_data.dist import SymbolDist, symbol_dist_from_json, symbol_dist_to_json


class Request(NamedTuple):
    symbols: SymbolDist
    batch_proportion: float

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
        }


class SymbolInfo(NamedTuple):
    dist: SymbolDist

    @classmethod
    def from_json(cls, d: dict) -> 'SymbolInfo':
        v = FieldValidator(cls, d)

        return SymbolInfo(
            dist=v.get('dist', dict, build=symbol_dist_from_json),
        )
