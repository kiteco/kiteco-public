# This is based on https://github.com/zomux/neuralcompressor, which is MIT-licensed (see LICENSE)

from typing import List

import os
import time

import numpy as np
import tensorflow as tf


RANDOM_SEED = 3


class EmbeddingCompressor(object):
    # TODO: If we want to get fancy, we can experiment with simulated annealing and lower the temperature as
    # the training progresses
    _TAU = 1.0
    _BATCH_SIZE = 64

    def __init__(self, n_codebooks: int, n_entries: int, model_path: str):
        """
        n_codebooks: number of codebooks (subcodes)
        n_entries: number of vectors in each codebook
        model_path: prefix for saving or loading the parameters
        """
        self.M = n_codebooks
        self.K = n_entries
        self._model_path = model_path

    def train(self, embed_matrix: np.ndarray, tensorboard_path: str, max_epochs: int = 200):
        """Train the model to compress `embed_matrix`"""
        assert len(embed_matrix.shape) == 2

        sw = tf.summary.FileWriter(tensorboard_path)

        vocab_size = embed_matrix.shape[0]
        valid_ids: List[int] = np.random.RandomState(RANDOM_SEED).randint(
            0, vocab_size, size=(self._BATCH_SIZE * 10,)).tolist()

        # Training
        with tf.Graph().as_default(), tf.Session() as sess:
            scalars = {
                name: tf.placeholder(tf.float32, shape=[], name=name) for name in [
                    "train_loss", "train_maxp", "valid_loss", "valid_maxp"]
            }
            summary_op = tf.summary.merge([tf.summary.scalar(name, ph) for name, ph in scalars.items()])

            with tf.variable_scope("Graph", initializer=tf.random_uniform_initializer(-0.01, 0.01)):
                word_ids_var, loss_op, train_op, maxp_op = self._build_training_graph(embed_matrix)
            # Initialize variables
            tf.global_variables_initializer().run()
            best_loss = 100000
            saver = tf.train.Saver()

            sw.add_graph(sess.graph)

            vocab_list = list(range(vocab_size))
            for epoch in range(max_epochs):
                start_time = time.time()
                train_loss_list = []
                train_maxp_list = []
                np.random.shuffle(vocab_list)
                for start_idx in range(0, vocab_size, self._BATCH_SIZE):
                    word_ids = vocab_list[start_idx:start_idx + self._BATCH_SIZE]
                    loss, _, maxp = sess.run(
                        [loss_op, train_op, maxp_op],
                        {word_ids_var: word_ids}
                    )
                    train_loss_list.append(loss)
                    train_maxp_list.append(maxp)

                # Print every epoch
                time_elapsed = time.time() - start_time

                # Validation
                valid_loss_list = []
                valid_maxp_list = []
                for start_idx in range(0, len(valid_ids), self._BATCH_SIZE):
                    word_ids = valid_ids[start_idx:start_idx + self._BATCH_SIZE]
                    loss, maxp = sess.run(
                        [loss_op, maxp_op],
                        {word_ids_var: word_ids}
                    )
                    valid_loss_list.append(loss)
                    valid_maxp_list.append(maxp)

                train_loss = np.mean(train_loss_list)
                train_maxp = np.mean(train_maxp_list)
                valid_loss = np.mean(valid_loss_list)
                valid_maxp = np.mean(valid_maxp_list)

                summary = sess.run(summary_op, {scalars[k]: v for k, v in {
                    "train_loss": train_loss,
                    "train_maxp": train_maxp,
                    "valid_loss": valid_loss,
                    "valid_maxp": valid_maxp,
                }.items()})
                sw.add_summary(summary, epoch)

                # Report
                report_token = ""
                if valid_loss <= best_loss * 0.999:
                    report_token = "*"
                    best_loss = valid_loss
                    saver.save(sess, self._model_path)
                print("[epoch{}] train_loss={:.4f} train_maxp={:.4f} valid_loss={:.4f} valid_maxp={:.4f} bps={:0f} {}".format(
                    epoch,
                    train_loss, train_maxp,
                    valid_loss, valid_maxp,
                    len(train_loss_list) / time_elapsed,
                    report_token
                ))
        print("Training Done")

    def export(self, embed_matrix: np.ndarray, prefix: str):
        """Export word codes and codebook for given embedding.

        Saves:
            <prefix>.codebook.npy: serialized numpy array containing codebooks (float32)
            <prefix>.codes.npy: serialized numpy array containing the codes (uint8)

        Args:
            embed_matrix: original embedding
            prefix: prefix of saving path
        """
        assert os.path.exists(self._model_path + ".meta")
        vocab_size = embed_matrix.shape[0]
        with tf.Graph().as_default(), tf.Session() as sess:
            with tf.variable_scope("Graph"):
                word_ids_var, codes_op, reconstruct_op = self._build_export_graph(embed_matrix)
            saver = tf.train.Saver()
            saver.restore(sess, self._model_path)

            # Dump codebook
            codebook_tensor = sess.graph.get_tensor_by_name('Graph/codebook:0')
            np.save(prefix + ".codebook", sess.run(codebook_tensor))

            # Dump codes
            all_codes = []
            vocab_list = list(range(embed_matrix.shape[0]))
            for start_idx in range(0, vocab_size, self._BATCH_SIZE):
                word_ids = vocab_list[start_idx:start_idx + self._BATCH_SIZE]
                codes = sess.run(codes_op, {word_ids_var: word_ids}).tolist()
                all_codes += codes
            all_codes = np.array(all_codes, dtype=np.uint8)
            np.save(prefix + ".codes", all_codes)

    def _gumbel_dist(self, shape, eps=1e-20) -> tf.Tensor:
        U = tf.random_uniform(shape, minval=0, maxval=1)
        return -tf.log(-tf.log(U + eps) + eps)

    def _gumbel_softmax(self, logits: tf.Tensor, temperature: float) -> tf.Tensor:
        """Compute gumbel softmax."""
        y = logits + self._gumbel_dist(tf.shape(logits))
        return tf.nn.softmax(y / temperature)

    def _encode(self, input_matrix: tf.Tensor, word_ids: tf.Tensor) -> (tf.Tensor, tf.Tensor):
        input_embeds = tf.nn.embedding_lookup(input_matrix, word_ids, name="input_embeds")

        M, K = self.M, self.K

        with tf.variable_scope("h"):
            h = tf.nn.tanh(tf.layers.dense(input_embeds, M * K/2, use_bias=True))
        with tf.variable_scope("logits"):
            logits = tf.layers.dense(h, M * K, use_bias=True)
            logits = tf.log(tf.nn.softplus(logits) + 1e-8)
        logits = tf.reshape(logits, [-1, M, K], name="logits")
        return input_embeds, logits

    def _build_export_graph(self, embed_matrix: np.ndarray) -> (tf.Tensor, tf.Tensor, tf.Tensor):
        """Export the graph for exporting codes and codebooks.

        Args:
            embed_matrix: numpy matrix of original embeddings
        """
        embed_size = embed_matrix.shape[1]

        input_matrix = tf.constant(embed_matrix, name="embed_matrix")
        word_ids = tf.placeholder(tf.int32, shape=[None], name="word_ids")

        # Define codebooks
        codebooks = tf.get_variable("codebook", [self.M * self.K, embed_size], dtype=tf.float32)

        # Coding
        input_embeds, logits = self._encode(input_matrix, word_ids)  # ~ (B, M, K)
        codes = tf.cast(tf.argmax(logits, axis=2), tf.int32)  # ~ (B, M)

        # Reconstruct
        offset = tf.range(self.M, dtype=tf.int32) * self.K
        codes_with_offset = codes + offset[None, :]

        selected_vectors = tf.gather(codebooks, codes_with_offset)  # ~ (B, M, H)
        reconstructed_embed = tf.reduce_sum(selected_vectors, axis=1)  # ~ (B, H)
        return word_ids, codes, reconstructed_embed

    def _build_training_graph(self, embed_matrix: np.ndarray) -> (tf.Tensor, tf.Tensor, tf.Tensor):
        """Export the training graph.

        Args:
            embed_matrix: numpy matrix of original embeddings
        """
        embed_size = embed_matrix.shape[1]

        # Define input variables
        input_matrix = tf.constant(embed_matrix, name="embed_matrix")
        word_ids = tf.placeholder(dtype=tf.int32, shape=[None], name="word_ids")

        # Define codebooks
        codebooks = tf.get_variable("codebook", [self.M * self.K, embed_size], dtype=tf.float32)

        # Encoding
        input_embeds, logits = self._encode(input_matrix, word_ids)  # ~ (B, M, K)

        # Discretization
        D = self._gumbel_softmax(logits, self._TAU)
        gumbel_output = tf.reshape(D, [-1, self.M * self.K])  # ~ (B, M * K)
        maxp = tf.reduce_mean(tf.reduce_max(D, axis=2))

        # Decoding
        y_hat = tf.matmul(gumbel_output, codebooks)

        # Define loss
        loss = 0.5 * tf.reduce_sum((y_hat - input_embeds)**2, axis=1)
        loss = tf.reduce_mean(loss, name="loss")

        # Define optimization
        tvars = tf.trainable_variables()
        grads = tf.gradients(loss, tvars)
        optimizer = tf.train.AdamOptimizer(0.0001)
        train_op = optimizer.apply_gradients(zip(grads, tvars), name="train_op")

        return word_ids, loss, train_op, maxp
