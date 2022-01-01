from typing import NamedTuple

from ..model.model import Config as BaseConfig


class Config(NamedTuple):
    test_fraction: float = 0.2
    batch_size: int = 10

    base_config: BaseConfig = BaseConfig()
