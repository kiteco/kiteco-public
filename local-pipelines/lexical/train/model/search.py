from typing import NamedTuple, Tuple, Callable, List, Dict

import tensorflow as tf

from .config import SearchConfig
from .util import shape_list


class SearchPlaceholders(object):
    def __init__(self, defaults: SearchConfig):
        with tf.name_scope('placeholders'):
            # shape [batch, vocab], 0's to filter vocab entry, 1 to include
            self.valid_prefix_ids: tf.Tensor = tf.compat.v1.placeholder(
                dtype=tf.int64, shape=[None, None], name='valid_prefix_ids',
            )

            # parameters for search
            self.minp: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(defaults.minp, tf.float32),
                shape=[], name='minp',
            )

            # sample from the topk logits
            self.topk: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(defaults.topk, tf.int64),
                shape=[], name='topk',
            )

            # width of the beam search
            self.width: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(defaults.width, tf.int64),
                shape=[], name='width',
            )

            # temperature scaling for lexical tokens
            self.inv_lexical_temperature: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(1.0/defaults.lexical_temperature, tf.float32),
                shape=[], name='inv_lexical_temperature',
            )

            # temperature scaling for ident tokens
            self.inv_ident_temperature: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(1.0/defaults.ident_temperature, tf.float32),
                shape=[], name='inv_ident_temperature',
            )

            # num lexical tokens in vocab
            self.num_lexical_tokens: tf.Tensor = tf.compat.v1.placeholder_with_default(
                input=tf.cast(defaults.num_lexical_tokens, tf.int64),
                shape=[], name='num_lexical_tokens',
            )

    def dict(self) -> Dict[str, tf.Tensor]:
        return {
            # self.minp.name: self.minp, // this gets pruned from the graph because its not used
            self.valid_prefix_ids.name: self.valid_prefix_ids,
            self.topk.name: self.topk,
            self.width.name: self.width,
            self.inv_lexical_temperature.name: self.inv_lexical_temperature,
            self.inv_ident_temperature.name: self.inv_ident_temperature,
            self.num_lexical_tokens.name: self.num_lexical_tokens
        }


class SearchResults(NamedTuple):
    results: tf.Tensor
    probs: tf.Tensor


class SearchParams(NamedTuple):
    phs: SearchPlaceholders
    depth: int
    # used for temperature scaling
    n_vocab: int
    batch: tf.Tensor
    # next_logits for the provided hypotheses
    # - the first parameter is the current search iteration, or -1 if this is the "init" call,
    #   when this is -1 then the second parameter is None.
    # - the second parameter is the current set of hypotheses, shape [batch, width, vocab],
    #   this is None if this is the "init" call to the search.
    # - the return value is the logits for the next possible expansion of each hypothesis,
    #   shape [batch, width, vocab]
    # NOTE: `search` below handles temperature scaling so client's should send in the raw logits.
    # TODO: better name?
    next_logits: Callable[[tf.Tensor], tf.Tensor]


def _add_results(sr: SearchResults, *, candidates, probs) -> SearchResults:
    """
    Pads sr.{results,probs} then appends canidates and probs and returns the results.
    Let num_hyps = width*depth + 1
    :param candidates: shape [batch, width, depth+1]
    :param probs: shape [batch, width, depth+2]
    :param sr: 
        - sr.results.shape == [batch, num_hyps, depth]
        - sr.probs.shape == [batch, num_hyps, depth+1]
    :return srr:
        - srr.results.shape == [batch, num_hyps + width, depth+2]
        - srr.probs.shape == [batch, num_hyps + width, depth+3]
    """
    # pad existing results and probs so we maintain proper shape
    num_pad = shape_list(candidates)[2] - shape_list(sr.results)[2]
    pad = -1 * tf.ones(shape_list(sr.results)[:-1] + [num_pad], dtype=tf.int64)
    results = tf.concat([sr.results, pad], axis=2, name='padded_results')

    num_pad = shape_list(probs)[2] - shape_list(sr.probs)[2]
    pad = tf.zeros(shape_list(sr.probs)[:-1] + [num_pad], dtype=tf.float32)
    results_probs = tf.concat([sr.probs, pad], axis=2, name='padded_results_probs')

    # add candidates from an iteration of the beam search
    results = tf.concat([results, candidates], axis=1, name='new_results')
    results_probs = tf.concat([results_probs, probs], axis=1, name='new_results_probs')

    return SearchResults(results=results, probs=results_probs)


def _select(sp: SearchParams, *, current_candidates, current_probs, logits) -> Tuple[tf.Tensor, tf.Tensor]:
    """Expand the current candidates based on logits for possible extensions.
    NOTE:
        - selection logic from https://arxiv.org/pdf/1701.03185.pdf
        - we use the "gumbel-max-k" trick to sample without replacement
            - https://github.com/tensorflow/tensorflow/issues/9260
            - https://timvieira.github.io/blog/post/2014/07/31/gumbel-max-trick/
        - must have width <= topk
    TODO:
        - topp filtering (e.g nucleus sampling)
        - minp filtering?
    :param current_candidates: shape [batch, width, depth]
    :param current_probs: shape [batch, width, depth+1]
    :param logits: shape [batch, width, vocab]
    :param topk: shape []
    :param width: shape []
    :return new_cands, new_probs:
        - new_cands.shape == [batch, width, depth+1]
        - new_probs.shape == 
    """
    batch = shape_list(logits)[0]

    # sample 1
    with tf.name_scope('sample_1'):
        # compute probs on the full vocab
        # shape [batch, width, vocab]
        probs = tf.nn.softmax(logits)

        # select topk, this truncates the distribution to avoid sampling from the tail
        # shape [batch, width, topk], each element of topk_idx is in [0,1,...,vocab-1]
        topk_logits, topk_idx = tf.nn.top_k(logits, tf.cast(sp.phs.topk, tf.int32))

        # Use the gumbel-max-k trick to sample <width> samples *without* replacement
        gumbel = -tf.math.log(-tf.math.log(tf.random.uniform(tf.shape(topk_logits), 0, 1)))

        # [batch, width, width], each element is in [0,1,...,topk-1]
        _, select_topk_idx = tf.nn.top_k(topk_logits + gumbel, tf.cast(sp.phs.width, tf.int32))

        # get original idxs
        # [batch, width, width], each element in [0,1,...,vocab-1]
        selected_idx = gather_elems_3d(topk_idx, select_topk_idx, name='selected_idx')

        # get probs for the samples from the full vocab probs
        # [batch, width, width]
        samples_probs = gather_elems_3d(probs, selected_idx, name='samples_probs')

    with tf.name_scope('build_possible_candidates'):
        # [batch, width, 1]
        last_probs = tf.expand_dims(current_probs[:, :, -1], axis=-1, name='last_probs')

        # probs for each new possible beam, shape [batch, width, width]
        probs = last_probs * samples_probs

        # [batch, width, depth, width]
        candidates = tf.tile(tf.expand_dims(current_candidates, axis=3), [1, 1, 1, sp.phs.width])
        cand_probs = tf.tile(tf.expand_dims(current_probs, axis=3), [1, 1, 1, sp.phs.width])

        # [batch, width, 1, width]
        expanded_selected = tf.expand_dims(tf.cast(selected_idx, dtype=tf.int64), axis=2)
        expanded_probs = tf.expand_dims(probs, axis=2)

        # [batch, width, depth+1, width]
        candidates = tf.concat([candidates, expanded_selected], axis=2)
          # [batch, width, depth+2, width] (extra 1 in the front)
        cand_probs = tf.concat([cand_probs, expanded_probs], axis=2)

        # [batch, width, width, depth+1]
        candidates = tf.transpose(candidates, [0,1,3,2])
        # [batch, width, width, depth+2]
        cand_probs = tf.transpose(cand_probs, [0,1,3,2])

        # [batch, width*width, depth+1]
        candidates = tf.reshape(candidates, [batch, -1, shape_list(candidates)[3]])
        # [batch, width*width, depth+2]
        cand_probs = tf.reshape(cand_probs, [batch, -1, shape_list(cand_probs)[3]])

    with tf.name_scope('sample_2'):
        # for each request (batch dimension) we want to sample
        # from the possible [width, width] beams, using the probs for each beam
        # and normalizing across all beams (for a particular request).
        # Use the gumbel-max-k trick to sample <width> samples *without* replacement

        # shape [batch, width*width]
        dist = tf.math.log(tf.reshape(probs, [batch, -1]))
        gumbel = -tf.math.log(-tf.math.log(tf.random.uniform(tf.shape(dist), 0, 1)))

        # shape [batch, width], elements in [0, ..., width*width - 1]
        _, samples = tf.nn.top_k(dist + gumbel, tf.cast(sp.phs.width, tf.int32))

        # shape [batch, width, depth+1]
        candidates = tf.gather(candidates, samples, batch_dims=1, axis=1)
        # shape [batch, width, depth+2]
        cand_probs = tf.gather(cand_probs, samples, batch_dims=1, axis=1)

    return candidates, cand_probs


def _expand(sr: SearchResults, sp: SearchParams,  *, candidates, probs, logits) -> Tuple[SearchResults, tf.Tensor, tf.Tensor]:
    """
    :param probs: has shape [batch, width, depth+1], maintains chained probabilities of the search candidates
                  so probs[:, :,-1] is the current probability of each candidate hypothesis.
    :param candidates: has shape [batch, width, depth]
    :param logits: has shape [width, vocab]
    """
    with tf.name_scope('select_candidates'):
        # [batch, width, depth+1], [batch, width, depth+2]
        candidates, probs = _select(sp, current_candidates=candidates, current_probs=probs, logits=logits)

    with tf.name_scope('add_results'):
        sr = _add_results(sr, candidates=candidates, probs=probs)

    return sr, candidates, probs


def _body(i: int, sr: SearchResults, sp: SearchParams, *, candidates, probs) -> Tuple[SearchResults, tf.Tensor, tf.Tensor]:
    with tf.name_scope('next_logits'):
        if i == -1:
            # explicitly pass in None for candidates to avoid clients trying to
            # use these results
            logits = sp.next_logits(-1, None)
        else:
            logits = sp.next_logits(i, candidates)

    with tf.name_scope('expand'):
        return _expand(sr, sp, candidates=candidates, probs=probs, logits=logits)


def _next_logits_fn(sp: SearchParams) -> Callable[[int, tf.Tensor], tf.Tensor]:
    with tf.name_scope('build_temperature_scaling'):
        # lexical tokens are at the beginning of the vocab
        temp_scaling = tf.compat.v2.where(
            tf.range(0, sp.n_vocab, dtype=tf.int64) < sp.phs.num_lexical_tokens,
            x=sp.phs.inv_lexical_temperature,
            y=sp.phs.inv_ident_temperature,
        )
    
    # grab closure of variables for safety
    client_next_logits = sp.next_logits
    valid_prefix_ids = sp.phs.valid_prefix_ids
    def next_logits(i: int, candidates: tf.Tensor) -> tf.Tensor:
        logits = client_next_logits(i, candidates)
        
        if i == -1:
            # NOTE: we do not do prefix normalization (like we do in the TFPredictor) because
            # we implicitly do this by masking the logits below. Once we take the softmax of the
            # logits, all the invalid tokens are set to prob 0 and the probabilities for the valid
            # tokens sum to 1, which accomplishes the same goal. The only difference is that
            # in the TFPredictor we also add a small regularizer to the renormalized probabilities
            # but we do not do that here since it doesn't seem to have a significant impact for
            # the large models.
            with tf.name_scope('mask_invalid_prefix_logits'):
                # valid_prefix_ids is a bit-vector with 1's set on ids we want to keep
                # and zeros otherwise. comes in with the shape [batch, vocab]
                # [batch, vocab] -> [batch, 1, vocab] to match logits
                # TODO: kind of nasty
                prefix_ids = tf.expand_dims(valid_prefix_ids, axis=1)

                # create ones_like logits, subtract prefix_ids to get the values to remove.
                # multiply by large number 1e10 so when we subtract, it will zero out the logits
                # that are invalid
                mask = tf.cast(1e10, tf.int64) * \
                    (tf.ones_like(logits, tf.int64) - prefix_ids)
                logits = tf.identity(logits - tf.cast(mask, tf.float32), name='masked_logits')

        with tf.name_scope('temperature_scaling'):
            return logits * temp_scaling

    return next_logits


def search(sp: SearchParams) -> SearchResults:
    """
    Performs search using the specified params.
    Let num_hyps = sp.width * sp.depth + 1
    :param sp: parameters defining the search
    :return sr:
        - sr.results.shape == [batch, num_hyps, depth+1], all hypotheses that were generated
          during the search. Padded with -1 if the hypothesis has length < depth+1
        - sr.probs.shape == [batch, num_hyps, depth+1], all of the chained probabilities for 
          each hypothesis generated during the search. Padded with 0 if the hypothesis has length < depth+1.
          More concretely, sr.probs[:,:, -1] is the probability of the full hypothesis (ignoring padding).
    """
    sp = SearchParams(
        phs=sp.phs,
        depth=sp.depth,
        n_vocab=sp.n_vocab,
        batch=sp.batch,
        next_logits=_next_logits_fn(sp)
    )

    with tf.name_scope('init'):
        # these are the initial results and candidates,
        # 1 hypothesis for each batch of length 0,
        # the prob for each initial hypothesis is 1
        # this is mostly to make the internal code the computes
        # the chained probabilities cleaner, we trim it below before returning.
        sr = SearchResults(
            results=tf.zeros([sp.batch, 1, 0], tf.int64),
            probs=tf.ones([sp.batch, 1, 1], tf.float32),
        )
        sr, candidates, probs = _body(-1, sr, sp, candidates=sr.results, probs=sr.probs)

    for i in range(sp.depth):
        with tf.name_scope(f'iter{i}'):
            sr, candidates, probs = _body(i, sr, sp,  candidates=candidates, probs=probs)

    with tf.name_scope('finalize_results'):
        # chop off leading ones from probs
        probs = sr.probs[:, :, 1:]
    
    return SearchResults(results=sr.results, probs=probs)


def gather_elems_3d(params: tf.Tensor, idxs: tf.Tensor, name: str = 'gather_elems_3d'):
    """
    params: [batch, row, col]
    idxs:   [batch, row, col_idx (N)]
    returns:[batch, row, col(col_idx) (N)]
    """
    # [batch, row, N] -> [batch, row, N, 1]
    idxs = tf.expand_dims(idxs, axis=-1)
    return tf.gather_nd(params, idxs, batch_dims=2, name=name)
