import numpy as np

import matplotlib.pyplot as plt
import seaborn as sns

import kite.ranking.tree as tree


def main():
    xs = np.random.uniform(-3, 3, 20)[:, np.newaxis]
    ys = np.sin(xs) + np.random.randn(*xs.shape) * 1e-2

    model = tree.fit_least_squares(xs, ys, learning_rate=1.)

    xs_validate = np.linspace(-3, 3, 100)
    ys_validate = np.sin(xs_validate)
    xs_validate = np.reshape(xs_validate, (100, 1))

    ys_predicted = np.array(list(map(model, xs_validate)))

    print('Predictions:\n', ys_predicted)

    plt.clf()
    plt.plot(xs, ys, '.', label='Data')
    plt.plot(xs_validate, ys_validate, '-', label='True')
    plt.plot(xs_validate, ys_predicted, '-', label='Predicted')
    plt.legend()
    plt.savefig('out/tree.pdf')


if __name__ == '__main__':
    main()
