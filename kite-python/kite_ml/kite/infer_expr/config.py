from typing import NamedTuple, Dict

from ..graph_encoder.ggnn import Config as GGNNConfig
from ..graph_encoder.embeddings import Config as EmbeddingConfig

from ..graph_data.graph import EdgeType

from ..infer_call.symbols import FuncInfos

from .attr_base import SymbolInfo as AttrBaseSymbolInfo

from ..infer_attr.symbols import SymbolInfo as AttrInfos

from ..infer_production.index import Index as ProductionIndex
from ..infer_production.config import Config as ProductionConfig

from ..utils.embeddings import CodebookConfig

from ..model.config import LossOpt, PoolingOpt

from ..asserts.asserts import FieldValidator


class Config(NamedTuple):
    ggnn: GGNNConfig = GGNNConfig(
        edge_set=[
            EdgeType.AST_CHILD,
            EdgeType.NEXT_TOKEN,
            EdgeType.DATA_FLOW,
            EdgeType.SCOPE,  # NOTE: this MUST to be included
        ],
        message_pooling=PoolingOpt.MAX,
        tie_fwd_bkwd_weights=True,
        use_edge_attention=True,
        separate_grus_per_step=False,
    )
    max_hops: int = 3
    embedding: EmbeddingConfig = EmbeddingConfig(
        subtoken_depth=150,
        subtoken_pooling=PoolingOpt.AVG,
        type_pooling=PoolingOpt.MAX,
        type_depth=150,
        type_codebook=CodebookConfig(n_codebooks=32, n_entries=64, enabled=True),
        subtoken_codebook=CodebookConfig(n_codebooks=32, n_entries=32, enabled=True),
    )
    production: ProductionConfig = ProductionConfig(
        depth=200,
        decouple_decoder_dim=False,
        loss=LossOpt.MAX_MARGIN,
        concat_context=False,
        codebook=CodebookConfig(n_codebooks=32, n_entries=64, enabled=True),
    )
    loss: LossOpt = LossOpt.MAX_MARGIN


class MetaInfo(NamedTuple):
    call: FuncInfos
    attr: AttrInfos
    production: ProductionIndex
    attr_base: AttrBaseSymbolInfo
    name_subtoken_index: Dict[str, int]
    type_subtoken_index: Dict[str, int]

    @classmethod
    def from_json(cls, d: dict) -> 'MetaInfo':
        v = FieldValidator(cls, d)

        return MetaInfo(
            call=v.get('call', dict, build=FuncInfos.from_json),
            attr=v.get('attr', dict, build=AttrInfos.from_json),
            production=v.get('production_index', dict, build=ProductionIndex.from_json),
            attr_base=v.get('attr_base', dict, build=AttrBaseSymbolInfo.from_json),
            name_subtoken_index=v.get_map('name_subtoken_index', str, int),
            type_subtoken_index=v.get_map('type_subtoken_index', str, int),
        )
