import unittest
import pytest

import numpy as np
import numdifftools

from .ranklearning import *

class TreeTest(unittest.TestCase):
        def test_listify(self):
                @listify
                def foo():
                        return range(5)
                self.assertListEqual(foo(), list(range(5)))

        def test_minmax_normalizer(self):
                a = np.arange(5).astype(float)
                b = np.arange(3).astype(float)
                nrm = minmax_normalizer(a)
                np.testing.assert_array_almost_equal(nrm(a), a/4.)
                np.testing.assert_array_almost_equal(nrm(b), b/4.)

        def test_meanstd_normalizer(self):
                a = np.arange(5).astype(float)
                b = np.arange(3).astype(float)
                nrm = meanstd_normalizer(a)
                np.testing.assert_array_almost_equal(nrm(a), (a-2.)/np.sqrt(2.))
                np.testing.assert_array_almost_equal(nrm(b), (b-2.)/np.sqrt(2.))

        def test_relations_from_ranks(self):
                ranks = [1, 2, 2, 5]
                relations = relations_from_ranks(ranks)
                self.assertListEqual(relations, [(0, 1), (0, 2), (0, 3), (1, 3), (2, 3)])

        def test_relations_from_relevance(self):
                relevance = [1, 3, 4, 3]
                relations = relations_from_scores(relevance)
                self.assertListEqual(relations, [(1, 0), (2, 0), (2, 1), (3, 0), (2, 3)])

        def test_rank(self):
                features = np.arange(5).astype(float)[:, None]
                ranks = rank(features, lambda x: x[..., 0])
                self.assertListEqual(ranks, [4, 3, 2, 1, 0])

        def test_hinge_loss(self):
                features = np.arange(5).astype(float)[:, None]
                relevance = np.arange(5)
                example_ids = np.arange(5)

                scorer = lambda x: x[..., 0]

                query = Query(None, features, relevance, example_ids)

                loss = HingeLoss()
                self.assertEqual(compute_dataset_loss([query], scorer, loss), 0)

                query.relations = [(0, 1)]
                self.assertEqual(compute_dataset_loss([query], scorer, loss), 2.)

                self.assertEqual(loss.gradient(1, 3), (-1., 1.))
                self.assertEqual(loss.gradient(3, 1), (0., 0.))

                si, sj = 5., 3.
                numeric_gradient = numdifftools.Derivative(lambda si: loss(si, sj))
                np.testing.assert_array_almost_equal(numeric_gradient(si), loss.gradient(si, sj)[0])

        def test_zero_one_loss(self):
                features = np.arange(5).astype(float)[:, None]
                relevance = np.arange(5)
                example_ids = np.arange(5)

                scorer = lambda x: x[..., 0]

                query = Query(None, features, relevance, example_ids)

                loss = ZeroOneLoss()
                query.relations = [(0, 1)]
                self.assertEqual(compute_dataset_loss([query], scorer, loss), 1.)

        def test_cross_entropy_loss(self):
                scorer = lambda x: x[..., 0]

                features = np.arange(2).astype(float)[:, None]
                relevance = np.arange(2)
                example_ids = np.arange(2)

                query = Query(None, features, relevance, example_ids)

                loss = CrossEntropyLoss()
                self.assertEqual(compute_dataset_loss([query], scorer, loss), np.log1p(np.exp(-1)))

                d = -1./(1. + np.exp(2))
                self.assertEqual(loss.gradient(3., 1.), (d, -d))
                d = -0.5
                self.assertEqual(loss.gradient(3., 3.), (d, -d))
                d = -1./(1 + np.exp(-2))
                self.assertEqual(loss.gradient(1., 3.), (d, -d))

                si, sj = 5., 3.
                numeric_gradient = numdifftools.Derivative(lambda si: loss(si, sj))
                np.testing.assert_array_almost_equal(numeric_gradient(si), loss.gradient(si, sj)[0])

        def test_rbf(self):
                self.assertAlmostEqual(Rbf(-4)(3, 5), np.exp(-16))

        def test_linear_scorer_evaluate(self):
                features = np.arange(12).reshape((6, 2))
                scorer = LinearScorer([0, 1])
                np.testing.assert_array_almost_equal(scorer(features), features[:, 1])

        @pytest.mark.skip('broken test; incorrect initialization of LinearRankLearner')
        def test_linear_rank_learner(self):
                features = np.arange(12).reshape((6, 2))
                relevance = np.arange(6)
                example_ids = np.arange(6)
                query = Query(None, features, relevance, example_ids)

                # we use psuedo relations here so that we can do manual calcuation (when needed) easier.
                relations = [(0, 1), (4, 2), (4, 0), (1, 5)]
                query.relations = relations

                def f(w):
                        loss = HingeLoss()
                        return compute_dataset_loss([query], LinearScorer(w), loss)

                loss = HingeLoss()
                w0 = np.asarray([0, 0], dtype=float)
                j_loss_wrt_w_numeric = numdifftools.Gradient(f)(w0)
                j_loss_wrt_w_analytic = LinearRankLearner(2).gradient(features, [], relations, None, loss)

                np.testing.assert_array_almost_equal(j_loss_wrt_w_numeric, j_loss_wrt_w_analytic)

        def test_kernel_scorer_evaluate(self):
                features = np.arange(12).reshape((6, 2))
                data = np.arange(8).reshape((4, 2))
                coefs = np.arange(4)
                scorer = KernelScorer(data, coefs)
                np.testing.assert_array_almost_equal(scorer(features), np.dot(np.inner(features, data), coefs))

        @pytest.mark.skip('broken; [ -2., -10.,  -9., -13., -17., -21.] vs [ -2., -10., -18., -26., -34., -42.]')
        def test_kernel_rank_learner(self):
                features = np.arange(12).reshape((6, 2))
                relevance = np.arange(6)
                example_ids = np.arange(6)
                query = Query(None, features, relevance, example_ids)

                # we use psuedo relations here so that we can do manual calcuation (when needed) easier.
                relations = [(0, 1), (4, 2), (4, 0), (1, 5)]
                query.relations = relations

                def f(coefs):
                        loss = HingeLoss()
                        return compute_dataset_loss([query], KernelScorer(features, coefs), loss)

                loss = HingeLoss()
                coefs0 = np.zeros(6)
                j_loss_wrt_w_numeric = numdifftools.Gradient(f)(coefs0)
                j_loss_wrt_w_analytic = KernelRankLearner(features).gradient(features, [], relations, None, loss)

                np.testing.assert_array_almost_equal(j_loss_wrt_w_numeric, j_loss_wrt_w_analytic)

        def test_tree_scorer_evaluate(self):
                features = np.arange(12).reshape((6, 2))
                targets = np.arange(6)

                scorer = TreeEnsembleScorer([])
                np.testing.assert_array_almost_equal(scorer(features), np.zeros(6))
                
                t = tree.fit_least_squares(features, targets, 1., max_depth=1)
                scorer.trees.append(t)
                np.testing.assert_array_almost_equal(scorer(features), [1, 1, 1, 4, 4, 4])

        def test_tree_rank_learner(self): 
                features = np.arange(12).reshape((6, 2))
                relevance = np.arange(6)
                example_ids = np.arange(6)
                query = Query(None, features, relevance, example_ids)

                # we use psuedo relations here so that we can do manual calcuation (when needed) easier.
                relations = [(0, 1), (4, 2), (4, 0), (1, 5)]
                query.relations = relations

                loss = CrossEntropyLoss(2) 

                learner = BoostedTreeLearner([query])
                raw_gradients_i, raw_gradients_j, raw_irs = learner.derivative_preprocess(
                        0, features, [], relations, None, loss) 

                j_loss_wrt_score_numeric = [0, 0, 1, 0, -2, 1] 
                j_loss_wrt_score_analytic = learner.gradient(
                        raw_gradients_i, raw_gradients_j, raw_irs, len(features), query.relations) 

                np.testing.assert_array_almost_equal(j_loss_wrt_score_numeric, j_loss_wrt_score_analytic)

                j_hessian_wrt_score_numeric = [2, 2, 1, 0, 2, 1]
                j_hessian_wrt_score_analytic = learner.hessian(
                        raw_gradients_i, raw_gradients_j, raw_irs, len(features), query.relations, loss.sigma) 

                np.testing.assert_array_almost_equal(j_hessian_wrt_score_numeric, j_hessian_wrt_score_analytic)

        def test_ndcg(self):
                labels = np.arange(10).astype(float)
                # we care only about the top 5 results
                ndcg = NDCG(5)

                # test score()
                for i in range(5):
                    label = 9 - i
                    self.assertEqual(ndcg.dcg(i, label), (np.power(2, label) - 1)/np.log(i + 2))

                # test score() for ranking below T
                for i in range(5, 10):
                    self.assertEqual(ndcg.dcg(i, 9 - i), 0)

                # test max_dcg
                self.assertEqual(ndcg.max_dcg(labels), sum(
                    (np.power(2, 9 - i) - 1) / np.log(i + 2) for i in range(5)))

                # test detal()
                # note that the positions in the test here are 0-based.
                # data[0] is currently at position 0
                # data[8] is currently at position 8
                delta = (np.power(2, 8) - 1) / np.log(2)
                self.assertEqual(ndcg.delta(labels[0], 0, labels[8], 8), delta)

                # data[0] is currently at position 8
                # data[8] is currently at position 0
                self.assertEqual(ndcg.delta(labels[0], 8, labels[8], 0), delta)

                # data[1] is currently at position 2
                # data[8] is currently at position 4
                delta = abs((np.power(2, 1) - np.power(2, 8)) / np.log(4)
                        + (np.power(2, 8) - np.power(2, 1)) / np.log(6))
                self.assertEqual(ndcg.delta(labels[1], 2, labels[8], 4), delta)

                # data[1] is currently at position 5
                # data[5] is currently at position 8
                self.assertEqual(ndcg.delta(labels[1], 5, labels[5], 8), 0)
