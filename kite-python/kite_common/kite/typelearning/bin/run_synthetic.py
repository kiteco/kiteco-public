import numpy as np

from kite.typelearning import typelearning


def main():
	np.random.seed(0)

	import_tree = {
		"ndarray": ["transpose", "sum", "shape"],
		"matrix": ["transpose", "sum", "shape", "trace"],
		"list": ["append", "extend", "index"],
		"deque": ["append", "appendleft", "extend", "extendleft"],
	}

	usages_by_func = {
		"zeros": {
			"my_ndarray": ["transpose"],
			"my_matrix": ["sum", "shape", "sum"]
		},
		"ones": {
			"my_ndarray_2": ["transpose"],
			"my_ndarray_3": ["sum", "shape", "sum"]
		},
		"makedeque": {
			"my_deque_1": ["append", "appendleft", "extend"],
			"my_deque_2": ["append", "extend", "extend"]
		},
		"deepcopy": {
			"my_list_1": ["append"],
			"my_deque_3": ["appendleft"],
			"my_list_2": ["index", "extend", "index"]
		},
		"filter": {
			"my_list_3": ["append", "append"],
			"my_list_4": ["append", "extend", "index"]
		}
	}

	typelearning.train(import_tree, usages_by_func)


if __name__ == "__main__":
	main()
