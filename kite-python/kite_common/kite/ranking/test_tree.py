import numpy as np
import unittest

from .tree import (
    DecisionTree,
    Node,
    drop,
    fit_newton,
    fit_least_squares,
    fit_skeleton,
    walk,
    weighted_variance,
)


class TreeTest(unittest.TestCase):
        def test_weighted_variance(self):
                a = np.array([1., 2., 3.])
                self.assertEqual(weighted_variance(a), 2.)

        def test_depth(self):
                n = Node(2, 1., Node(0, -.5, None, None), None)
                self.assertEqual(n.depth(), 2)

        def test_walk(self):
                n1 = Node(0, 0, None, None)
                n2 = Node(0, 0, None, None)
                n3 = Node(0, 0, n1, n2)
                n4 = Node(0, 0, n3, None)
                self.assertSequenceEqual(list(walk(n3)), [n3, n1, n2])
                self.assertSequenceEqual(list(walk(n1)), [n1])
                self.assertSequenceEqual(list(walk(n4)), [n4, n3, n1, n2])

        def test_drop(self):
                n = Node(2, 1., Node(0, -.5, 0, 1), 2)
                self.assertEqual(drop([-1, 0, 5], n), 2)

        def test_tojson(self):
                n = Node(2, 1., Node(0, -.5, 0, 1), 2)
                t = DecisionTree(n, [10, 20, 30], 3)
                expected = {
                    'outputs': [10, 20, 30],
                    'nodes': [{
                        'feature_index': 2,
                        'right_is_leaf': True,
                        'right_child': 2,
                        'left_child': 1,
                        'left_is_leaf': False,
                        'threshold': 1.0
                    }, {
                        'feature_index': 0,
                        'right_is_leaf': True,
                        'right_child': 1,
                        'left_child': 0,
                        'left_is_leaf': True,
                        'threshold': -0.5
                    }],
                    'feature_size': 3,
                    'depth': 2}
                self.assertDictEqual(t.to_json(), expected)

        def test_evaluate(self):
                n = Node(2, 1., Node(0, -.5, 0, 1), 2)
                t = DecisionTree(n, [10, 20, 30], 3)
                self.assertEqual(t([-1, 0, 5]), 30)

        def test_fit_skeleton_depth_1(self):
                data = np.arange(24).reshape((6, 4))
                targets = np.arange(6)
                root, bins = fit_skeleton(data, targets, max_depth=1)
                self.assertEqual(root.depth(), 1)
                self.assertEqual(len(bins), 2)

        def test_fit_newton_steps(self):
                data = np.arange(12).reshape((6, 2))
                targets = np.arange(6)
                hessians = np.arange(6)

                tree = fit_newton(data, targets, 1., newton_step=hessians, max_depth=1)
                self.assertSequenceEqual(tree.outputs, [1, 1])

        def test_fit_skeleton_depth_4(self):
                data = np.arange(240).reshape((60, 4))
                targets = np.arange(60)
                root, bins = fit_skeleton(data, targets, max_depth=4)
                self.assertEqual(root.depth(), 4)
                self.assertEqual(len(bins), 16)

        def test_fit_least_squares(self):
                data = np.arange(24).reshape((6, 4))
                targets = np.arange(6)
                t = fit_least_squares(data, targets, 1., max_depth=2)
                self.assertEqual(t.root.depth(), 2)
                self.assertEqual(len(t.outputs), 4)
