from typing import Tuple, Optional

import tensorflow as tf

from .reduce import safe_reduce_mean


def segment_topk(preds: tf.Tensor, segment_ids: tf.Tensor, k: int,
                 prefix: Optional[str]=None) -> Tuple[tf.Tensor, tf.Tensor, tf.Tensor]:
        """
        segment version of tf.nn.top_k, returns three tensors, all of rank 1 with the same
        shape, (probs, idxs, ids) where:
        - probs containts the probabilities of at most the top k elements for each segment, a segment that has
        less than k elements will just contain all elements in sorted order.
        - idxs contains the original index for each element in probs, e.g we have that preds[idxs[i]] == probs[i]
        - ids contains the segment ids for each entry in probs, e.g ids[i] = j implies entry i in probs is for segment j.

        :param preds: predictions, rank 1
        :param segment_ids: segment  ids for preds, segment_ids.shape == preds.shape
        :param k: top k results for each segment
        :param prefix: prefix for operation names
        :return (probs, idxs, ids): all returned tensors are rank 1,
        and probs.shape == idxs.shape == ids.shape.
        """
        def opname(suffix: str) -> str:
            if not prefix:
                return 'segment_topk_' + suffix
            return prefix + '_segment_topk_' + suffix

        with _valid_segment(preds, segment_ids, 'preds', opname('')):
            #
            # Sort predictions and segment ids first by prediction probability, then resort into the
            # proper segment, tracking the original rows for each prediction probability as we go
            #

            # sort predictions from highest to lowest, the idxs are the labels for the predictions
            # since they correspond to the row (label) in pred where the value came from
            idxs_by_preds: tf.Tensor = tf.contrib.framework.argsort(preds, direction='DESCENDING',
                                                                    stable=True,
                                                                    name=opname('idxs_by_preds'))

            preds_by_preds: tf.Tensor = tf.gather(preds, idxs_by_preds, name=opname('preds_by_preds'))

            # sort segment ids to make sure we track the segment that each prediction came from
            ids_by_preds: tf.Tensor = tf.gather(segment_ids, idxs_by_preds, name=opname('ids_by_preds'))

            # next we sort the segment ids, from lowest to highest, that were already sorted by predictions
            # because we know that the predictions
            # are sorted absolutely so within each segment the predictions are also sorted,
            # so sorting the segment ids allows us to sort the predictions within each segment
            # this works because stable argsort chooses the lowest index value when it encounters values that are equal
            idxs_by_ids_by_preds: tf.Tensor = tf.contrib.framework.argsort(ids_by_preds, direction='ASCENDING',
                                                                           stable=True,
                                                                           name=opname('idxs_by_ids_by_preds'))

            ids_by_ids_by_preds: tf.Tensor = tf.gather(
                ids_by_preds, idxs_by_ids_by_preds, name=opname('ids_by_ids_by_preds')
            )

            preds_by_ids_by_preds: tf.Tensor = tf.gather(
                preds_by_preds, idxs_by_ids_by_preds, name=opname('preds_by_ids_by_preds')
            )

            # map sorted labels from first step to the appropriate segments after we have sorted by segment ids
            row_by_ids_by_preds: tf.Tensor = tf.gather(
                idxs_by_preds, idxs_by_ids_by_preds, name=opname('row_by_ids_by_preds')
            )

            #
            # now we need to limit our results to k for each segment
            #

            # get number of elements in each segment
            _, _, elems_per_segment = tf.unique_with_counts(segment_ids, name=opname('elems_per_segment'))

            # choose min between number of elements per segment and k
            # [num_segments, 1]
            min_elems_per_segment_or_k: tf.Tensor = tf.expand_dims(tf.minimum(elems_per_segment, k), axis=1,
                                                                   name=opname('min_elems_per_segment_or_k'))

            max_elems: tf.Tensor = tf.reduce_max(min_elems_per_segment_or_k, name=opname('max_elems'))

            # need conditional for empty batches since reduce_max returns a negative number
            # if min_elems_per_segment_or_k is empty
            max_elems = tf.cond(
                max_elems < tf.constant(0), true_fn=lambda: 0, false_fn=lambda: max_elems,
                name=opname('max_elems_or_0'),
            )

            num_segments: tf.Tensor = tf.shape(elems_per_segment, name=opname('num_segments'))[0]

            # create offset matrix for each segment
            # [1, max_elems]
            max_offsets: tf.Tensor = tf.expand_dims(tf.range(max_elems, name=opname('max_offsets')),
                                                    axis=0, name=opname('max_offsets_expanded'))

            # [num_segments, max_elems]
            offsets_per_segment: tf.Tensor = tf.tile(max_offsets, [num_segments, 1],
                                                     name=opname('offsets_per_segment'))

            # mask out invalid offsets for each segment
            # [num_segments, max_elems]
            mask: tf.Tensor = tf.cast(offsets_per_segment < min_elems_per_segment_or_k,
                                      offsets_per_segment.dtype, name=opname('mask'))

            offsets_per_segment_masked: tf.Tensor = tf.multiply(offsets_per_segment, mask,
                                                                name=opname('offsets_per_segment_masked'))

            # [num_segments, 1]
            segment_start_idxs: tf.Tensor = tf.expand_dims(
                tf.cumsum(elems_per_segment, exclusive=True, name=opname('segment_start_idxs')),
                axis=1, name=opname('segment_start_idxs_expanded'))

            # create matrix of offsets for each segment, including duplicates
            # [num_segments * max_elems]
            rows_to_select: tf.Tensor = tf.reshape(segment_start_idxs + offsets_per_segment_masked, [-1],
                                                   name=opname('rows_to_select'))

            # de-dupe rows
            rows_to_select_deduped, _ = tf.unique(rows_to_select, name=opname('rows_to_select_deduped'))

            final_rows: tf.Tensor = tf.gather(row_by_ids_by_preds, rows_to_select_deduped, name=opname('final_rows'))

            final_preds: tf.Tensor = tf.gather(preds_by_ids_by_preds, rows_to_select_deduped, name=opname('final_preds'))

            final_ids: tf.Tensor = tf.gather(ids_by_ids_by_preds, rows_to_select_deduped, name=opname('final_ids'))

            return final_preds, final_rows, final_ids


def segment_maxmargin_loss(logits: tf.Tensor, labels: tf.Tensor,
                           segment_ids: tf.Tensor, corrupted: tf.Tensor, name: str) -> tf.Tensor:
    """
    Compute segmented max margin loss:
    SO link: /questions/37689632/max-margin-loss-in-tensorflow
    http://web.stanford.edu/class/cs224n/lectures/lecture4.pdf

    Typically the segments correspond to different samples in a batch, and the segmented
    corrupted labels are the corrupted labels for each sample in the batch.

    :param logits: logits to compute the max margin loss over, must be rank 1

    :param labels: labels for the batch, shape [batch size], dtype={tf.int32, tf.int64}

    :param segment_ids: segment ids for the corrupted labels,
    e.g segment_ids[i] = j implies that corrupted[i] is a corrupted label for
    segment j. segment_ids.shape == corrupted.shape, dtype={tf.int32, tf.int64}

    :param corrupted: corrupted labels for all the samples in the batch,
    shape [num corrupted samples], dytpe={tf.int32, tf.int64}

    :param name: name for the resulting tensor

    :return: shape []
    """
    def opname(suffix: str) -> str:
        return name + '_max_margin_' + suffix

    with _valid_segment(corrupted, segment_ids, 'corrupted', opname('')):
        # [batch size]
        true_scores: tf.Tensor = tf.gather(logits, labels, name=opname('true_scores'))

        # broadcast true scores to each corrupted segment in batch
        # [num corrupted samples]
        true_scores = tf.gather(true_scores, segment_ids, name=opname('true_scores_broadcast'))

        # [num corrupted samples]
        corrupted_scores: tf.Tensor = tf.gather(logits, corrupted, name=opname('corrupted_scores'))

        # to make type checker happy
        one: tf.Tensor = tf.constant(1.)

        # [num corrupted samples]
        maxes: tf.Tensor = tf.maximum(0., one - true_scores + corrupted_scores, name=opname('maxes'))

        # need to reduce mean across each segment, then across segments
        means: tf.Tensor = tf.segment_mean(maxes, segment_ids, name=opname('segmented_means'))

        # shape []
        return safe_reduce_mean(means, 0., name=name)


def segment_softmax(logits: tf.Tensor, segment_ids: tf.Tensor, name: str) -> tf.Tensor:
    """
    Compute segmented softmax:
    - N = len(logits)
    - B = number of segments
    - i indexes over segments
    - x = logits
    - x_ij = jth element in segment i
    - m_i = max_j(x_ij)
    - y_ij = exp( x_ij - m_i)
    - z_ij = y_ij / sum_j(y_ij)

    Assumes:
    - logits has shape [None]
    :param logits: must be rank 1, logits for the elements of each segment
    :param segment_ids: must be rank 1, segment ids for the logits
    :param name: name of the resulting operation
    :return val: val.shape == logits.shape
    """
    def opname(suffix: str) -> str:
        return name + '_segment_softmax_' + suffix

    with _valid_segment(logits, segment_ids, 'logits', opname('')):
        # [B]
        maxes: tf.Tensor = tf.segment_max(logits, segment_ids, name=opname('maxes'))

        # distribute max back out to each logit based on segment
        # [N]
        maxes_expanded: tf.Tensor = tf.gather(maxes, segment_ids, name=opname('maxes_expanded'))

        # subtract max and exponentiate
        # [N]
        numerator: tf.Tensor = tf.exp(logits - maxes_expanded, name=opname('numerator'))

        # sum along segments to compute denominators
        # [B]
        denominator: tf.Tensor = tf.segment_sum(numerator, segment_ids, name=opname('denominator'))

        # distribute denominator back out to size N
        # [N]
        denominator = tf.gather(denominator, segment_ids, name=opname('denominator_expanded'))

        return tf.div(numerator, denominator, name=name)


def segment_accuracy(pred: tf.Tensor, labels: tf.Tensor, segment_ids: tf.Tensor, topk: int) -> tf.Tensor:
    """
    Compute segmented accuracy at k.

    Typically the segments correspond to different samples in a batch

    :param pred: prediction probabilities to compute the accuracy with respect to

    :param labels: labels for the batch, shape [batch size], dtype={tf.int32, tf.int64}

    :param segment_ids: segment ids for the predictions,
    e.g segment_ids[i] = j implies that pred[i] is a prediction probability for
    segment j. segment_ids.shape == pred.shape, dtype={tf.int32, tf.int64}

    :param topk: accuracy to use for the second tensor

    :return: 0D tensor representing the accuracy at k
    """

    def opname(suffix: str) -> str:
        return 'segment_accuracy_' + suffix

    _, idxs, sample_ids = segment_topk(
        pred, segment_ids, topk, opname('top_{}'.format(topk)),
    )

    if topk == 1:
        # idxs has shape [num calls in batch]
        return safe_reduce_mean(
            tf.cast(tf.equal(labels, idxs), dtype=tf.float32),
            0., opname('accuracy'),
        )

    # expand true labels to same shape as idxs via sample ids
    true_labels: tf.Tensor = tf.gather(
        labels, sample_ids, opname('true_labels'),
    )

    atk: tf.Tensor = tf.cast(
        tf.equal(true_labels, idxs), tf.float32, opname('at_{}'.format(topk)),
    )

    # sum over sample first, then mean over batch, this works because exactly
    # one label per task will match so this sum is equivalent to a logical or over each sample
    return safe_reduce_mean(
        tf.segment_sum(atk, sample_ids), 0., opname('acc_at_{}'.format(topk)),
    )


def normalize_segment_ids(segment_ids: tf.Tensor, unique_idxs: Optional[tf.Tensor], name: Optional[str]) -> tf.Tensor:
    """
    Normalize the provided segment ids to be in a contiguos range between 0,...,len(set(segment_ids)).
    NOTE: segment_ids MUST be sorted and non negative. TODO: check this or just sort automatically?
    :param segment_ids: 1D tensor, len N
    :param unique_idxs: optional, 1D tensor idxs of len N, as returned by tf.unique(segment_ids)[1]
    :param name: optional name
    :return: 1D tensor W of len N, such that W[0] == 0, W[len(W)-1] == len(set(segment_ids)), and
            W is sorted in ascending order.
    TODO: unit test!!!!
    TODO: add some checks?
    """
    def opname(suffix: str) -> str:
        return name + '_' + suffix
    if unique_idxs is not None:
        _, unique_idxs = tf.unique(segment_ids, name=opname('uniqueify'))
    return tf.identity(unique_idxs, name=name)


def _valid_segment(params: tf.Tensor, segment_ids: tf.Tensor, params_name: str, name: str):
    def opname(suffix: str) -> str:
        return name + '_valid_segment_' + suffix

    assert_shape = tf.assert_equal(
        tf.shape(segment_ids), tf.shape(params),
        message='segment_ids and {} must have same shape'.format(params_name),
        name=opname('assert_rank'),
    )

    assert_rank = tf.assert_rank(
        segment_ids, 1,
        message='segment_ids must be rank 1',
        name=opname('assert_rank'),
    )

    return tf.control_dependencies([assert_shape, assert_rank])
