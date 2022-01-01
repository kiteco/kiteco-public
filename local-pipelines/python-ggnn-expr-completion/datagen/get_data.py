import argparse
import datetime
import json
import logging
import pickle
import time
import shutil

from kite.graph_data.data_feeder import EndpointDataFeeder
from kite.graph_data.session import RequestInit
from kite.graph_data.graph_feed import GraphFeedConfig

from kite.infer_expr.config import MetaInfo, Config

from kite.infer_call.request import Request as CallRequest, KwargRequest, ArgTypeRequest, ArgPlaceholderRequest
from kite.infer_expr.request import Request as ExprRequest
from kite.infer_expr.attr_base import Request as AttrBaseRequest
from kite.infer_attr.request import Request as AttrRequest


logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


def get_filename(cur_sample: int, total: int, timestamp: int) -> str:
    n_digits = len(str(total))
    format_str = "{{:0{}d}}".format(n_digits) + "-of-{}-{}.pickle"
    return format_str.format(cur_sample, total, timestamp)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--endpoint', type=str, default='http://localhost:3039')
    parser.add_argument('--random_seed', type=int)
    parser.add_argument('--batch', type=int, default=10)
    parser.add_argument('--samples', type=int, default=1000, help='number of samples to generate')
    parser.add_argument('--meta_info', type=str)
    parser.add_argument('--out_dir', type=str, default='data')
    parser.add_argument('--samples_per_file', type=int, default=500)
    parser.add_argument('--max_samples', type=int)
    parser.add_argument('--attr_base_proportion', type=float)
    parser.add_argument('--attr_proportion', type=float)
    parser.add_argument('--call_proportion', type=float)
    parser.add_argument('--arg_type_proportion', type=float)
    parser.add_argument('--kwarg_name_proportion', type=float)
    parser.add_argument('--arg_placeholder_proportion', type=float)
    args = parser.parse_args()

    meta_info = MetaInfo.from_json(json.load(open(args.meta_info, 'r')))

    config = Config()

    req = RequestInit(
        config=GraphFeedConfig(edge_set=config.ggnn.edge_set),
        random_seed=args.random_seed,
        num_batches=args.batch,
        max_hops=config.max_hops,
        name_subtoken_index=meta_info.name_subtoken_index,
        type_subtoken_index=meta_info.type_subtoken_index,
        production_index=meta_info.production,
        expr=ExprRequest(
            max_samples=args.max_samples,
            call=CallRequest(
                symbols=meta_info.call.dist,
                batch_proportion=args.call_proportion,
            ),
            attr=AttrRequest(
                symbols=meta_info.attr.dist,
                batch_proportion=args.attr_proportion,
                parents=meta_info.attr.parents,
            ),
            attr_base=AttrBaseRequest(
                symbols=meta_info.attr_base.dist,
                batch_proportion=args.attr_base_proportion,
            ),
            arg_type=ArgTypeRequest(
                symbols=meta_info.call.dist,
                batch_proportion=args.arg_type_proportion,
            ),
            kwarg_name=KwargRequest(
                symbols=meta_info.call.dist,
                keywords=meta_info.call.keywords,
                batch_proportion=args.kwarg_name_proportion,
            ),
            arg_placeholder=ArgPlaceholderRequest(
                symbols=meta_info.call.dist,
                batch_proportion=args.arg_placeholder_proportion,
            )
        ),
    )

    logging.info("will write {} samples to {}, random seed = {}".format(
        args.samples, args.out_dir, args.random_seed))

    feeder = EndpointDataFeeder(args.endpoint, req)

    try:
        tmp_filename = None
        filename = None
        file = None
        file_samples = 0
        start = None

        n_names = 0
        n_production = 0

        def finish_file():
            file.close()
            shutil.move(tmp_filename, filename)

            end = datetime.datetime.now()
            logging.info(
                "sample {}: saved {} with {} samples ({} name, {} production), took {}".format(
                    i, filename, args.samples_per_file, n_names, n_production, end - start
                ))

        for i in range(args.samples):
            if not file or file_samples >= args.samples_per_file:
                if file:
                    finish_file()
                    file_samples = 0

                ts = int(time.time() * 1000)
                filename = "{}/{}".format(args.out_dir, get_filename(i, args.samples, ts))
                tmp_filename = "{}.part".format(filename)
                file = open(tmp_filename, 'wb')

                start = datetime.datetime.now()
                logging.info("writing to {}".format(tmp_filename))

            sample = feeder.next()
            pickle.dump(sample, file)
            n_names += len(sample.data.expr.infer_name.prediction_nodes)
            n_production += len(sample.data.expr.infer_production.prediction_nodes)
            file_samples += 1

        if file_samples > 0:
            finish_file()

    finally:
        feeder.stop()


if __name__ == "__main__":

    main()
