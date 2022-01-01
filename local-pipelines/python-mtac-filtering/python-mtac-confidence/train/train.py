from typing import Dict

import argparse
import datetime
import logging
import os
import tensorflow as tf

from kite.infer_mtac_conf.config import Config
from kite.infer_mtac_conf.file_feeder import FileFeederSplit
from kite.infer_mtac_conf.model import Model
from kite.infer_mtac_conf.train import train_logistic_model

from kite.utils.save import purge_dir, save_model, save_frozen_model

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')

DEFAULT_TRAINDATA_PATH = 'src/github.com/kiteco/kiteco/local-pipelines/python-completion-mixing/out/traindata.json'


def save(model: Model, sess: tf.Session, out_dir: str, frozen_model_path: str):
    inputs = model.placeholders().dict()
    outputs: Dict[str, tf.Tensor] = {
        model.pred().name: model.pred(),
    }

    save_model(sess, out_dir, inputs=inputs, outputs=outputs)
    save_frozen_model(sess, frozen_model_path, list(outputs.keys()))


def main():
    ts = str(datetime.datetime.now())

    default_traindata_path = os.path.join(os.environ['GOPATH'], DEFAULT_TRAINDATA_PATH)

    parser = argparse.ArgumentParser()
    parser.add_argument('--traindata', type=str, default=default_traindata_path)
    parser.add_argument('--out_dir', type=str, default='out/{}/model'.format(ts))
    parser.add_argument('--frozen_model', type=str, default='out/{}/mix_model.frozen.pb'.format(ts))

    args = parser.parse_args()

    purge_dir(args.out_dir)
    config = Config()

    model = Model(config)
    split = FileFeederSplit(args.traindata, config.test_fraction)

    with tf.Session() as sess:
        sess.run(tf.global_variables_initializer())

        start = datetime.datetime.now()
        train_logistic_model(sess, model, split.train_feeder(), split.val_feeder())
        end = datetime.datetime.now()

        logging.info('Done training, took {0}'.format(end - start))

        save(model, sess, args.out_dir, args.frozen_model)


if __name__ == "__main__":
    main()



