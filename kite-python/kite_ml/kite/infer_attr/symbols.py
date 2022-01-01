from typing import Dict, List, NamedTuple

from ..asserts.asserts import FieldValidator

from ..graph_data.dist import SymbolDist, symbol_dist_from_json


class SymbolInfo(NamedTuple):
    dist: SymbolDist
    parents: Dict[str, str]  # map from child symbol to its parent

    @classmethod
    def from_json(cls, d: dict) -> 'SymbolInfo':
        v = FieldValidator(cls, d)
        return SymbolInfo(
            dist=v.get('dist', dict, build=symbol_dist_from_json),
            parents=v.get_map('parents', str, str),
        )
