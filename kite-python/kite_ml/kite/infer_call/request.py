from typing import NamedTuple, Dict, List

from ..graph_data.dist import SymbolDist, symbol_dist_to_json


class Request(NamedTuple):
    symbols: SymbolDist
    batch_proportion: float

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
        }


class KwargRequest(NamedTuple):
    symbols: SymbolDist
    keywords: Dict[str, List[str]]
    batch_proportion: float

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
            'keywords': self.keywords,
        }


class ArgTypeRequest(NamedTuple):
    symbols: SymbolDist
    batch_proportion: float

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
        }


class ArgPlaceholderRequest(NamedTuple):
    symbols: SymbolDist
    batch_proportion: float

    def to_json(self) -> dict:
        return {
            'symbols': symbol_dist_to_json(self.symbols),
            'batch_proportion': self.batch_proportion,
        }
