from json import JSONEncoder
from typing import Any, Dict, List

from ..asserts.asserts import Assert, Validator
from .config import Config
from .constants import Constants


class RawFeatures(object):
    class Keys(object):
        LAST_SIBLING = 'LastSibling'
        PARENT_NODE = 'ParentNode'
        FIRST_TOKEN = 'FirstToken'
        REL_INDENT = 'RelIndent'
        PREVIOUS = 'Previous'
        FIRST_CHAR = 'FirstChar'
        PREVIOUS_KEYWORDS = 'PreviousKeywords'

        @classmethod
        def validators(cls, config: Config) -> List[Validator]:
            return [
                Validator(cls.LAST_SIBLING, int, Assert.categorical(high=Constants.N_NODES, low=-1)),
                Validator(cls.PARENT_NODE, int, Assert.categorical(high=Constants.N_NODES)),
                Validator(cls.FIRST_TOKEN, int, Assert.categorical(high=Constants.N_TOKENS)),
                Validator(cls.REL_INDENT, int, Assert.categorical(high=Constants.N_REL_INDENT)),
                Validator(cls.PREVIOUS, List,
                          Assert.chain(Assert.has_at_least_len(config.lookback),
                                       Assert.map(Assert.categorical(high=Constants.N_TOKENS)))),
                Validator(cls.PREVIOUS_KEYWORDS, List,
                          Assert.chain(Assert.has_at_least_len(Constants.N_KEYWORDS),
                                       Assert.map(Assert.categorical(high=2)))),
                Validator(cls.FIRST_CHAR, int, Assert.categorical(high=Constants.N_PREFIXES, low=-1)),
            ]

    def __init__(self, d: Dict[str, Any]):
        self.last_sibling: int = d[self.Keys.LAST_SIBLING]
        self.parent_node: int = d[self.Keys.PARENT_NODE]
        self.first_token: int = d[self.Keys.FIRST_TOKEN]
        self.rel_indent: int = d[self.Keys.REL_INDENT]
        self.prev_tokens: List[int] = d[self.Keys.PREVIOUS]
        self.first_char: int = d[self.Keys.FIRST_CHAR]
        self.previous_keywords: List[int] = d[self.Keys.PREVIOUS_KEYWORDS]

    @staticmethod
    def assert_valid(d: Dict[str, Any], config: Config):
        Assert.valid(d, RawFeatures.__name__, RawFeatures.Keys.validators(config))


class RawRecord(object):
    class Keys(object):
        FEATURES = 'Features'
        IS_KEYWORD = 'IsKeyword'
        KEYWORD_CATEGORY = 'KeywordCategory'

        @classmethod
        def validators(cls, config: Config) -> List[Validator]:
            return [
                Validator(cls.FEATURES, dict, lambda _, d: RawFeatures.assert_valid(d, config)),
                Validator(cls.IS_KEYWORD, bool),
                Validator(cls.KEYWORD_CATEGORY, int, Assert.categorical(high=Constants.N_KEYWORDS + 1)),
            ]

    def __init__(self, d: Dict[str, Any]):
        self.features = RawFeatures(d[self.Keys.FEATURES])
        self.is_keyword: bool = d[self.Keys.IS_KEYWORD]
        self.keyword_cat: int = d[self.Keys.KEYWORD_CATEGORY]

    @staticmethod
    def assert_valid(d: Dict[str, Any], config: Config):
        Assert.valid(d, RawRecord.__name__, RawRecord.Keys.validators(config))
