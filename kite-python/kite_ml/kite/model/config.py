from enum import Enum


class LossOpt(Enum):
    CROSS_ENTROPY = 'cross_entropy'
    MAX_MARGIN = 'max_margin'


class PoolingOpt(Enum):
    SUM = 1
    MAX = 2
    AVG = 3
