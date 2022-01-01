from typing import NamedTuple
from kite.asserts.asserts import FieldValidator


class Config(NamedTuple):
    n_vocab: int = 3082
    n_ctx: int = 128
    n_embd: int = 180
    n_head: int = 6
    n_layer: int = 6
    n_prediction_slots: int = 10
    model_type: str = 'lexical'
    n_langs: int = 23
    n_full_embd: int = 180

    @classmethod
    def from_json(cls, d: dict) -> 'Config':
        v = FieldValidator(cls, d)
        return Config(
            n_vocab=v.get('n_vocab', int),
            n_embd=v.get('n_embd', int),
            n_ctx=v.get('n_ctx', int),
            n_head=v.get('n_head', int),
            n_layer=v.get('n_layer', int),
            n_prediction_slots=v.get('n_prediction_slots', int),
            model_type=d.get('model_type', 'lexical'),  # use raw dict for backwards compat
            n_langs=v.get('n_langs', int),
            n_full_embd=v.get('n_full_embd', int),
        )
    
    def down_project_embds(self) -> bool:
        return self.n_embd != self.n_full_embd

def update(config: Config, d: dict) -> 'Config':
    original = dict(config._asdict())
    for k in d:
        original[k] = d[k]
    return Config.from_json(original)


class SearchConfig(NamedTuple):
    window: int = 400
    topk: int = 10
    topp: float = 1.0
    minp: float = 0.0
    width: int = 5
    depth: int = 5
    prefix_regularization: float = 0.05
    ident_temperature: float = 1.0
    lexical_temperature: float = 1.0
    num_lexical_tokens: int = 0

    @classmethod
    def from_json(cls, d: dict) -> 'SearchConfig':
        v = FieldValidator(cls, d)
        return SearchConfig(
            window=v.get('Window', int),
            topk=v.get('TopK', int),
            topp=v.get_float('TopP'),
            minp=v.get_float('MinP'),
            width=v.get('BeamWidth', int),
            depth=v.get('Depth', int),
            prefix_regularization=v.get_float('PrefixRegularization'),
            ident_temperature=v.get_float('IdentTemperature'),
            lexical_temperature=v.get_float('LexicalTemperature'),
            num_lexical_tokens=v.get('NumLexicalTokens', int),
        )

