from typing import NamedTuple, Dict

from ..graph_data.dist import SymbolDist, symbol_dist_to_json


class Request(NamedTuple):
    symbols: SymbolDist
    batch_proportion: float
    parents: Dict[str, str]

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
            'parents': self.parents,
        }
