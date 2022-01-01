import argparse
import logging
import os
import random
import shutil

import numpy as np
import tensorflow as tf
from kite.infer_attribute.data import DataFeeder, LabelTaxonomy
from kite.infer_attribute.model import GGNN, Config

VALIDATION_INTERVAL = 10
MAX_ITER = 1000 * 1000 * 10
SAVE_INTERVAL = 20000  #

MODEL_NAME = 'refactor_pr'
TENSORBOARD_PATH = '/data/kite/tensorboard/' + MODEL_NAME
MODEL_SAVE_PATH = '/data/kite/models/' + MODEL_NAME


def main():
    # for reproducibility, set random seed
    random.seed(243)
    np.random.seed(243)

    purge_dir(TENSORBOARD_PATH)
    args = parse_args()

    label_taxonomy = LabelTaxonomy.from_json()

    train_feeder = DataFeeder(label_taxonomy, args.local)

    validation_feeder = train_feeder  # temporary hack until we get a "validation" endpoint.

    logging.info('model name {}'.format(MODEL_NAME))

    config = Config(label_taxonomy, max_reduce_model=False)
    model = GGNN(config)

    train_iter = train_feeder.__iter__()
    val_feeder = validation_feeder.__iter__()
    summary_writer = tf.summary.FileWriter(TENSORBOARD_PATH)

    with tf.Session() as sess:
        sess.run(tf.global_variables_initializer())
        for step in range(MAX_ITER):

            if step % SAVE_INTERVAL == 0:
                try_rmdir(MODEL_SAVE_PATH)
                model.save(sess, export_dir=MODEL_SAVE_PATH)

            logging.info('step {:d}'.format(step))
            train_feed = next(train_iter)
            train_res = model.step(sess, feed=train_feed)

            # loss only equals 0.0 if the class is out-of-taxonomy or when there's only 1 class, don't log it
            if train_res['logloss'] != 0.0:
                summary_writer.add_summary(train_res['summary'], step)
                logging.info('TRAIN logloss {:f}'.format(train_res["logloss"]))

            if step % VALIDATION_INTERVAL == 0:  # loss only equals 0.0 if the class is out-of-taxonomy
                val_feed = next(val_feeder, step)
                val_res = model.step(sess, feed=val_feed, train=False)
                if val_res['logloss'] != 0.0:
                    logging.info('VAL logloss {:f}'.format(val_res["logloss"]))
                    summary_writer.add_summary(val_res['summary'], step)


def purge_dir(path):
    try:
        shutil.rmtree(path)
    except OSError:
        pass
    try:
        os.makedirs(path)
    except OSError:
        pass


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('--local', type=int, default=0)
    return parser.parse_args()


def try_rmdir(path):
    try:
        shutil.rmtree(path)
    except OSError:
        pass


if __name__ == '__main__':
    main()
