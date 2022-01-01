import tensorflow as tf


class SoftmaxClassifier(object):
    # x should have a shape of [batch size, encoded feature size]
    def __init__(self, x: tf.Tensor, n_categories: int, scope: str):
        with tf.name_scope(scope):
            self.labels: tf.Tensor = tf.placeholder(tf.int64, [None], name='labels')  # shape: [batch size]
            # one-hot encode the class label
            y = tf.cast(tf.one_hot(self.labels, n_categories), tf.float32)

            enc_feature_size = int(x.shape[1])

            # Set model weights
            self.weights: tf.Tensor = tf.Variable(tf.zeros([enc_feature_size, n_categories]), name='weights')
            self.biases: tf.Tensor = tf.Variable(tf.zeros([n_categories]), name='biases')

            self.logits: tf.Tensor = tf.nn.softmax(tf.matmul(x, self.weights) + self.biases, axis=1, name='logits')

            correct_prediction = tf.equal(tf.argmax(self.logits, 1), tf.argmax(y, 1))
            self.accuracy: tf.Tensor = tf.reduce_mean(tf.cast(correct_prediction, tf.float32))

            # Some elements of the logits easily collapse to zero at the beginning, so add a bit of an epsilon to avoid
            # -inf logs
            # TODO(damian): perhaps we can achieve decent results by initializing the weights intelligently
            # instead of resorting to this
            self.loss: tf.Tensor = tf.reduce_mean(-tf.reduce_sum(y * tf.log(self.logits + 1e-10), axis=1))
            self.optimizer: tf.Operation = tf.train.AdamOptimizer().minimize(self.loss)
