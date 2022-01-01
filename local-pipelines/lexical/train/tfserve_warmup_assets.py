import numpy as np
import tensorflow as tf
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import prediction_log_pb2

from model.config import Config

import argparse
import json
import os

NUM_RECORDS = 10

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--config', type=str)
    parser.add_argument('--out_assets_extra', type=str)
    parser.add_argument('--model', type=str)
    args = parser.parse_args()

    config = Config.from_json(json.load(open(args.config, 'r')))
    assets_extra = os.path.join(args.out_assets_extra, "assets.extra")

    print("using config:", config)
    print("writing to:", assets_extra)

    os.mkdir(assets_extra)

    context_size = config.n_ctx-100
    with tf.io.TFRecordWriter(os.path.join(assets_extra, "tf_serving_warmup_requests")) as writer:
        req = predict_pb2.PredictRequest()
        req.model_spec.name = args.model
        req.model_spec.signature_name = 'serving_default'
        req.inputs['context_mask'].CopyFrom(tf.make_tensor_proto([[1]*context_size], tf.int64))
        req.inputs['valid_prefix_ids'].CopyFrom(tf.make_tensor_proto([[1]*config.n_vocab], tf.int64))

        log = prediction_log_pb2.PredictionLog(
            predict_log=prediction_log_pb2.PredictLog(request=req))

        for r in range(NUM_RECORDS):
            context = list(np.random.choice(range(config.n_vocab-1), context_size)+1) # range(n)-1, +1 to avoid 0 token, which is SOF
            req.inputs['context'].CopyFrom(tf.compat.v1.make_tensor_proto([context], tf.int64))
            log = prediction_log_pb2.PredictionLog(
                predict_log=prediction_log_pb2.PredictLog(request=req))
            writer.write(log.SerializeToString())




if __name__ == "__main__":
    main()
