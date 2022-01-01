import tensorflow as tf


def safe_cross_entropy(probs: tf.Tensor, labels: tf.Tensor, name: str) -> tf.Tensor:
    return tf.cond(
        tf.equal(tf.size(probs), 0),
        true_fn=lambda: tf.constant(0.),
        false_fn=lambda: tf.reduce_mean(-tf.log(tf.gather(probs, labels))),
        name=name,
    )
