from typing import NamedTuple, List, Dict

from ..asserts.asserts import FieldValidator


class Production(NamedTuple):
    id: str
    children: List[str]

    @classmethod
    def from_json(cls, d: dict) -> 'Production':
        v = FieldValidator(cls, d)
        return Production(
            id=v.get('id', int),
            children=v.get_list('children', int),
        )

    def to_json(self) -> dict:
        return {
            'id': self.id,
            'children': self.children,
        }


class Index(NamedTuple):
    productions: Dict[str, Production]
    indices: Dict[str, int]

    @classmethod
    def from_json(cls, d: dict) -> 'Index':
        v = FieldValidator(cls, d)
        return Index(
            productions=v.get_map('productions', str, dict, val_build=Production.from_json),
            indices=v.get_map('indices', str, int),
        )

    def to_json(self) -> dict:
        return {
            'productions': {k: v.to_json() for k, v in self.productions.items()},
            'indices': self.indices,
        }

    def vocab(self) -> int:
        return len(self.indices)
