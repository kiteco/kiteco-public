from typing import NamedTuple

from ..model.config import LossOpt
from ..utils.embeddings import CodebookConfig


class Config(NamedTuple):
    depth: int
    decouple_decoder_dim: bool
    loss: LossOpt
    concat_context: bool
    codebook: CodebookConfig
