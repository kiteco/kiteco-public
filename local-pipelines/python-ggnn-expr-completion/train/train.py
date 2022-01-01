from typing import Dict

import argparse
import datetime
import json
import logging
import numpy as np
import random
import tensorflow as tf
import os

from kite.graph_data.file_feeder import FileDataFeeder
from kite.graph_data.data_feeder import EndpointDataFeeder
from kite.graph_data.sync_feeder import SyncDataFeeder
from kite.graph_data.session import RequestInit
from kite.graph_data.graph_feed import GraphFeedConfig

from kite.model.model import TrainInputs, Config as BaseConfig, AdamTrainer, DataFeeder

from kite.utils.save import purge_dir, save_model, save_frozen_model

from kite.infer_expr.config import MetaInfo, Config
from kite.infer_expr.model import Model

from kite.infer_call.request import Request as CallRequest, KwargRequest, ArgTypeRequest, ArgPlaceholderRequest
from kite.infer_expr.request import Request as ExprRequest
from kite.infer_expr.attr_base import Request as AttrBaseRequest
from kite.infer_attr.request import Request as AttrRequest

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


def read_info(filename: str) -> Dict[str, float]:
    with open(filename, 'r') as f:
        return json.load(f)


def save(model: Model, sess: tf.Session, outdir: str, frozen_model_path: str):
    inputs = model.placeholders_dict()

    outputs = model.outputs_dict()

    save_model(
        sess,
        outdir,
        inputs=inputs,
        outputs=outputs,
    )

    save_frozen_model(
        sess,
        frozen_model_path,
        list(outputs.keys()),
    )


def main():
    random.seed(243)
    np.random.seed(243)

    parser = argparse.ArgumentParser()
    parser.add_argument('--meta_info', type=str)
    parser.add_argument('--out_dir', type=str)
    parser.add_argument('--frozen_model', type=str)
    parser.add_argument('--checkpoint_path', type=str)
    parser.add_argument('--tensorboard', type=str)
    parser.add_argument('--steps', type=int)

    # in dir for syncer or data saved to disk
    parser.add_argument('--in_dir', type=str)

    # saved data coordinated with the syncer
    parser.add_argument('--use_synced_data', type=bool, default=False)
    parser.add_argument('--syncer_endpoint', type=str)

    # saved data from disk
    parser.add_argument('--data_gen_local', type=bool, default=False)

    # optional: load pre-trained model from dir
    parser.add_argument('--load_model', type=str, default="")

    # graph-server-specific parameters
    parser.add_argument('--endpoint', type=str)
    parser.add_argument('--batch', type=int)
    parser.add_argument('--max_samples', type=int)
    parser.add_argument('--attr_base_proportion', type=float)
    parser.add_argument('--attr_proportion', type=float)
    parser.add_argument('--call_proportion', type=float)
    parser.add_argument('--arg_type_proportion', type=float)
    parser.add_argument('--kwarg_name_proportion', type=float)
    parser.add_argument('--arg_placeholder_proportion', type=float)

    args = parser.parse_args()

    purge_dir(args.tensorboard)
    purge_dir(args.out_dir)

    logging.info('writing tensorboard results to {0}'.format(args.tensorboard))

    meta_info = MetaInfo.from_json(read_info(args.meta_info))

    config = Config()

    model = Model(config, meta_info, compressed=False)

    trainer = AdamTrainer(model, BaseConfig(steps=args.steps, learning_rate=1e-5, skip_grad_summaries=True))

    if args.data_gen_local:
        logging.info('using file data feeder with {}'.format(args.in_dir))
        feeder = FileDataFeeder(args.in_dir)
    elif args.use_synced_data:
        logging.info('using sync data feeder with {} at {}'.format(args.in_dir, args.syncer_endpoint))
        feeder = SyncDataFeeder(args.in_dir, args.syncer_endpoint)
    else:
        req = RequestInit(
            config=GraphFeedConfig(edge_set=config.ggnn.edge_set),
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
        feeder = EndpointDataFeeder(args.endpoint, req)

    try:
        sw = tf.summary.FileWriter(args.tensorboard)

        with tf.Session() as sess:
            start = datetime.datetime.now()

            if args.load_model != "":
                saver = tf.train.Saver()

                # Load the most recent model from a checkpoint path, and get the current global step
                ckpt = tf.train.get_checkpoint_state(os.path.dirname(args.load_model))
                if ckpt and ckpt.model_checkpoint_path:
                    saver.restore(sess, ckpt.model_checkpoint_path)

                # Get the step from the filename. Kinda hacky.
                # TODO: use global_step variable instead.
                starting_step = int(os.path.basename(ckpt.model_checkpoint_path).split('-')[1]) + 1

            else:
                sess.run(tf.global_variables_initializer())
                starting_step = 0

            ti = TrainInputs(session=sess, train_feeder=feeder, val_feeder=feeder,
                             summary_writer=sw, checkpoint_save_path=args.checkpoint_path,
                             summary_interval=50, validation_interval=30,
                             starting_step=starting_step, checkpoint_interval=int(1e5))

            trainer.train(ti)
            end = datetime.datetime.now()
            logging.info('Done training, took {0}'.format(end-start))

            save(model, sess, args.out_dir, args.frozen_model)
    finally:
        feeder.stop()


if __name__ == '__main__':
    main()
