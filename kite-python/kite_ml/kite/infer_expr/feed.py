from typing import NamedTuple

from ..graph_data.graph_feed import GraphFeed
from ..infer_name.feed import Feed as NameModelFeed
from ..infer_production.feed import Feed as ProductionModelFeed
from ..graph_encoder.expansion_graph import TrainFeed as ExpansionGraphFeed
from ..asserts.asserts import FieldValidator

from .config import MetaInfo, Config


class RawTrainSample(NamedTuple):
    context_graph: GraphFeed
    infer_name: NameModelFeed
    infer_production: ProductionModelFeed
    expansion_graph: ExpansionGraphFeed

    @classmethod
    def from_json(cls, d: dict) -> 'RawTrainSample':
        v = FieldValidator(cls, d)
        return RawTrainSample(
            context_graph=v.get('context_graph', dict, build=GraphFeed.from_json),
            infer_name=v.get('infer_name', dict, build=NameModelFeed.from_json),
            infer_production=v.get('infer_production', dict, build=ProductionModelFeed.from_json),
            expansion_graph=v.get('expansion_graph', dict, build=ExpansionGraphFeed.from_json),
        )


class Feed(NamedTuple):
    context_graph: GraphFeed
    infer_name: NameModelFeed
    infer_production: ProductionModelFeed
    expansion_graph: ExpansionGraphFeed
    num_production_samples: float
    num_name_samples: float
    num_nodes_context_graph: float
    num_nodes_expansion_graph: float

    @classmethod
    def from_raw(cls, raw: RawTrainSample) -> 'Feed':
        return Feed(
            context_graph=raw.context_graph,
            infer_name=raw.infer_name,
            infer_production=raw.infer_production,
            expansion_graph=raw.expansion_graph,
            num_production_samples=float(raw.infer_production.batch_size()),
            num_name_samples=float(raw.infer_name.batch_size()),
            num_nodes_context_graph=float(len(set(raw.context_graph.node_subtokens.sample_ids))),
            num_nodes_expansion_graph=float(raw.expansion_graph.num_nodes()),
        )

    def assert_valid(self, config: Config, info: MetaInfo):
        self.context_graph.assert_valid(config.ggnn.edge_set,
                                        len(info.type_subtoken_index), len(info.name_subtoken_index))

        self.infer_name.assert_valid()

        self.infer_production.assert_valid()

        self.expansion_graph.assert_valid(config.ggnn.edge_set,
                                          len(info.type_subtoken_index), len(info.name_subtoken_index))
