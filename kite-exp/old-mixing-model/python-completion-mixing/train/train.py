from typing import Dict

import argparse
import datetime
import logging
import os
import tensorflow as tf

from kite.completion_mix.config import Config
from kite.completion_mix.file_feeder import Batcher, FileFeederSplit
from kite.completion_mix.model import Model

from kite.model.model import TrainInputs, Config as BaseConfig, AdamTrainer

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
    parser.add_argument('--tensorboard', type=str, default='data/{}/'.format(ts))
    parser.add_argument('--checkpoint_path', type=str, default='out/{}/ckpt'.format(ts))
    parser.add_argument('--steps', type=int, default=int(1e3))
    args = parser.parse_args()

    purge_dir(args.tensorboard)
    purge_dir(args.out_dir)

    logging.info('writing tensorboard results to {0}'.format(args.tensorboard))

    config = Config(base_config=BaseConfig(steps=args.steps))

    split = FileFeederSplit(args.traindata, config.test_fraction)
    train_feeder = Batcher(split.train_feeder(), config.batch_size)
    val_feeder = Batcher(split.val_feeder(), config.batch_size)

    model = Model(config)
    trainer = AdamTrainer(model, config.base_config)

    sw = tf.summary.FileWriter(args.tensorboard)

    with tf.Session() as sess:
        start = datetime.datetime.now()
        ti = TrainInputs(session=sess, train_feeder=train_feeder, val_feeder=val_feeder,
                         summary_writer=sw, checkpoint_save_path=args.checkpoint_path,
                         summary_interval=5, validation_interval=30, checkpoint_interval=int(1e5))
        sess.run(tf.global_variables_initializer())
        trainer.train(ti)

        end = datetime.datetime.now()
        logging.info('Done training, took {0}'.format(end - start))

        save(model, sess, args.out_dir, args.frozen_model)


if __name__ == "__main__":
    main()

