from typing import NamedTuple, Dict, List

from ..asserts.asserts import FieldValidator

from ..graph_data.dist import SymbolDist, symbol_dist_from_json


class FuncInfo(NamedTuple):
    kwarg_names: List[str]

    @classmethod
    def from_json(cls, d: dict) -> 'FuncInfo':
        v = FieldValidator(cls, d)
        return FuncInfo(
            kwarg_names=v.get('kwarg_names', list, build=list),
        )


class FuncInfos(NamedTuple):
    dist: SymbolDist
    keywords: Dict[str, List[str]]

    @classmethod
    def from_json(cls, d: dict) -> 'FuncInfos':
        v = FieldValidator(cls, d)

        infos = v.get_map('infos', str, dict, val_build=FuncInfo.from_json)

        keywords = dict()
        for sym, info in infos.items():
            keywords[sym] = info.kwarg_names
        return FuncInfos(
            dist=v.get('dist', dict, build=symbol_dist_from_json),
            keywords=keywords,
        )
