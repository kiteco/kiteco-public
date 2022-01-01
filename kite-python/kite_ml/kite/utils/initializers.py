import numpy as np


def glorot_init(vocab: int, depth: int) -> np.ndarray:
    """
    Implements http://proceedings.mlr.press/v9/glorot10a/glorot10a.pdf
    :param vocab:
    :param depth:
    :return: float32 tensor of shape [vocab, depth]
    """
    initialization_range = np.sqrt(6.0 / float(vocab + depth))
    uni = np.random.uniform(low=-initialization_range,
                            high=initialization_range, size=(vocab, depth))
    return uni.astype(np.float32)


def randn_init(vocab: int, depth: int, scale=1.) -> np.ndarray:
    randn = scale * np.random.randn(vocab, depth)
    return randn.astype(np.float32)
