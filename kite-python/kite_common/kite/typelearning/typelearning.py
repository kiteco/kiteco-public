import sys
import json
import gzip
import collections
import numpy as np
from pprint import pprint

from kite.emr import io
from kite.ioutils import stream


def row_normalize(x):
	"""
	Scale a matrix such that each row sums to one
	"""
	return x / x.sum(axis=1)[:, None]


def col_normalize(x):
	"""
	Scale a matrix such that each column sums to zero
	"""
	return x / x.sum(axis=0)


def logsumexp(x):
	"""
	Compute log(sum(exp(x))) avoiding over/underflow
	"""
	x = np.asarray(x)
	m = x.max()
	return np.log(np.sum(np.exp(x - m))) + m


def logsumexp_rowwise(x):
	"""
	Compute row-wise log(sum(exp(x))) avoiding over/underflow
	"""
	x = np.asarray(x)
	m = x.max(axis=1)[:, None]
	return np.log(np.sum(np.exp(x - m), axis=1)) + m


def logsumexp_colwise(x):
	"""
	Compute col-wise log(sum(exp(x))) avoiding over/underflow
	"""
	x = np.asarray(x)
	m = x.max(axis=0)
	return np.log(np.sum(np.exp(x - m), axis=0)) + m


def logsumexp_normalize(x):
	"""
	Add a constant to x such that logsumexp(x) = 1
	"""
	x = np.asarray(x)
	return x - logsumexp(x)


def logsumexp_row_normalize(x):
	"""
	Add a row-wise constant to x such that logsumexp_rowwise(x) = 0
	"""
	x = np.asarray(x)
	return x - logsumexp_rowwise(x)[:, None]


def logsumexp_col_normalize(x):
	"""
	Add a col-wise constant to x such that logsumexp_colwise(x) = 0
	"""
	x = np.asarray(x)
	return x - logsumexp_colwise(x)


def lastpart(x):
	"""
	Get the last identifier in a dotted expression
	"foo.bar.baz" -> "baz"
	"foo.bar" -> "bar"
	"foo" -> "foo"
	"""
	pos = x.rfind(".")
	if pos == -1:
		return x
	else:
		return x[pos+1:]


def logdistr_str(distr):
	"""
	Convert a set of log probabilities to an ordinary probability distribution, then return
	a string representation of that distribution.
	"""
	keys, logps = map(list, zip(*distr.items()))
	ps = np.exp(logsumexp_normalize(logps))
	indices = np.argsort(ps)[::-1][:5]
	names = list(map(lastpart, keys))
	return " ".join("%s:%.5f" % (names[i], ps[i]) for i in indices if not names[i].startswith("_"))


class Parameters(object):
	"""
	Represents estimates for the parameters we are attempting to learn.
	"""
	def __init__(self, types, funcs, attrs, vars, usages_by_func, seed_func_table, seed_attr_table, seed_var_type_distr):
		"""
		Initialize the optimization with data and seed estimates for each variable
		"""
		self.types = types   # fully qualified names of all known types (list of strings)
		self.funcs = funcs   # fully qualified names of all known functions (list of strings)
		self.attrs = attrs   # names of all known attributes (list of strings)
		self.vars = vars     # names of all tracked variables (list of strings)
		self.func_table = seed_func_table          # 2D array of size NUM_FUNCTIONS x NUM_TYPES
		self.attr_table = seed_attr_table          # 2D array of size NUM_ATTRIBUTES x NUM_TYPES
		self.var_type_distr = seed_var_type_distr  # 2D array of size NUM_VARIABLES x NUM_TYPES


class Learner(object):
	"""
	Performs expectation maximization steps to learn a mapping from function to returntype frequencies and a
	mapping from types to attribute frequencies.

	seed is an initial guess for the model parameters (an instance of Parameters)

	usages_by_func is the information from import exploration
	"""
	def __init__(self, seed, usages_by_func):
		self.cur = seed
		self.usages_by_func = usages_by_func

	def step(self):
		"""
		Perform one expectation maximization iteration
		"""
		# E step: compute probability distribution over types for each variable
		# Note that P(type | func, attrs) = P(type | func) \prod P(attr_i | type)
		self.cur.var_type_distr = np.zeros((len(self.cur.vars), len(self.cur.types)))
		for funcname, usages in self.usages_by_func.items():
			for varname, attrs in usages.items():
				self.cur.var_type_distr[self.cur.vars.index(varname)] = self.cur.func_table[self.cur.funcs.index(funcname)].copy()
				for attr in attrs:
					self.cur.var_type_distr[self.cur.vars.index(varname)] *= self.cur.attr_table[self.cur.attrs.index(attr)]

		self.cur.var_type_distr = row_normalize(self.cur.var_type_distr + 1e-8)
		assert np.all(self.cur.var_type_distr > 0)

		# M step pt 1: compute MAP estimate for func_table
		self.cur.func_table = np.zeros((len(self.cur.funcs), len(self.cur.types)))
		for funcname, usages in self.usages_by_func.items():
			for varname in usages.keys():
				self.cur.func_table[self.cur.funcs.index(funcname)] += self.cur.var_type_distr[self.cur.vars.index(varname)]

		self.cur.func_table = row_normalize(self.cur.func_table + 1e-8)
		assert np.all(self.cur.func_table > 0)

		# M step pt 2: compute MAP estimate for attr_table
		self.cur.attr_table = np.zeros((len(self.cur.attrs), len(self.cur.types)))
		for funcname, usages in self.usages_by_func.items():
			for varname, attrs in usages.items():
				for attr in attrs:
					self.cur.attr_table[self.cur.attrs.index(attr)] += self.cur.var_type_distr[self.cur.vars.index(varname)]

		self.cur.attr_table = col_normalize(self.cur.attr_table + 1e-8)
		assert np.all(self.cur.attr_table > 0)


def candidate_returntypes(usages, types_by_attr, max_candidates=10):
	"""
	Generate N candidate returntypes for the given function
	"""
	score_by_type = collections.defaultdict(float)
	for varname, attrs in usages.items():
		for attr in attrs:
			for typename in types_by_attr[attr]:
				score_by_type[typename] += 1

	topn = sorted(score_by_type.keys(), key=lambda t: score_by_type[t], reverse=True)[:max_candidates]
	return {t: score_by_type[t] for t in topn}


def train(attrs_by_type, usages_by_func, num_iters=5, seed_func_table=None, seed_attr_table=None):
	"""
	Given data about the attributes on various types and usage data relating functions to
	the attributes accessed on their returntypes, estimate the returntypes for each function,
	and also the frequency with which each attribute is accessed.
	"""
	# Make a list of all the attributes
	attributes = set()
	for typename, attrs in attrs_by_type.items():
		attributes.update(attrs)
	attributes = sorted(attributes)

	# Filter out the unrecognized usages
	num_attrs_total = 0
	num_attrs_dropped = 0
	num_vars_dropped = 0
	num_funcs_dropped = 0
	filtered_usages_by_func = {}
	for func, usages in usages_by_func.items():
		filtered_usages = {}
		for varname, attrs in usages.items():
			filtered_attrs = [attr for attr in attrs if attr in attributes]
			num_attrs_total += len(attrs)
			num_attrs_dropped += len(attrs) - len(filtered_attrs)
			if filtered_attrs:
				filtered_usages[varname] = filtered_attrs
			else:
				num_vars_dropped += 1
		if filtered_usages:
			filtered_usages_by_func[func] = filtered_usages

	print("Dropped %d unrecognized attributes (of %d total)" % (num_attrs_dropped, num_attrs_total))
	print("  and dropped %d empty variables (of %d total)" % (num_vars_dropped, sum(map(len, usages_by_func.values()))))
	print("  and dropped %d empty functions (of %d total)" % (num_funcs_dropped, len(attrs_by_type)))
	usages_by_func = filtered_usages_by_func

	types = sorted(attrs_by_type.keys())
	funcs = sorted(usages_by_func.keys())
	vars = sorted([x for v in usages_by_func.values() for x in v.keys()])

	# Initialize the func/type table
	if seed_func_table is None:
		# Construct the reverse mapping from attributes to types
		types_by_attr = collections.defaultdict(list)
		for typename, attrs in attrs_by_type.items():
			for attr in attrs:
				types_by_attr[attr].append(typename)

		# Generate candidates for each function
		candidates_by_func = {}
		seed_func_table = np.zeros((len(funcs), len(types)))
		for func, usages in usages_by_func.items():
			candidates = candidate_returntypes(usages, types_by_attr)
			candidates_by_func[func] = candidates
			for typename, p in candidates.items():
				seed_func_table[funcs.index(func), types.index(typename)] = p

		seed_func_table = row_normalize(seed_func_table)

	# Initialize the type/attr table
	if seed_attr_table is None:
		seed_attr_table = np.zeros((len(attributes), len(types)))
		for typename, attrs in attrs_by_type.items():
			for attr in attrs:
				seed_attr_table[attributes.index(attr), types.index(typename)] = 1.

		seed_attr_table = col_normalize(seed_attr_table)

	# Construct the seed from which to optimize
	seed = Parameters(types, funcs, attributes, vars, usages_by_func, seed_func_table, seed_attr_table, seed_var_type_distr=None)

	# Start optimization
	learner = Learner(seed, usages_by_func)
	for step in range(num_iters):
		print("\nSTEP %d" % step)
		learner.step()

	return learner.cur


def load_usages(path, pkgs):
	num_vars = 0
	usages_by_func = collections.defaultdict(dict)
	with open(path) as f:
		for key, val in io.read_as_json(f):
			name = val['Identifier']
			assert key == name

			if not any(name.startswith(pkg+".") for pkg in pkgs):
				continue
			
			attrs = [x["Identifier"] for x in val["Attributes"]]
			usages_by_func[name]["var%08d" % num_vars] = attrs
			num_vars += 1

	return usages_by_func


def load_import_tree(path, pkgs=None):
	funcs = []
	attrs_by_type = {}  # type -> attrs
	func_aliases = {}   # name -> canonical name

	with gzip.open(path) as f:
		for obj in stream.loadjson(f):
			name, cl, members, aliases = obj['canonical_name'], obj['classification'], obj['members'], obj['names']
			func_aliases[name] = name

			if pkgs is not None and not any(name.startswith(pkg+".") for pkg in pkgs):
				continue

			if cl == 'function':
				funcs.append(name)
				for alias in aliases:
					func_aliases[alias] = name
			elif cl == 'type':
				# Do not model constructors as functions - just treat them deterministically
				attrs_by_type[name] = members

	return attrs_by_type, funcs, func_aliases


def print_labelled_matrix(m, rowlabels, collabels):
	"""
	Print a matrix with text labels on both axes
	"""
	assert m.shape == (len(rowlabels), len(collabels))
	maxrowlabel = max(len(label) for label in rowlabels)

	hfmt = (" " * (maxrowlabel+1)) + " ".join(["%%%ds" % max(len(label), 8) for label in collabels])
	fmt = ("%%%ds " % maxrowlabel) + " ".join(["%%%d.3f" % max(len(label), 8) for label in collabels])

	print(hfmt % tuple(collabels))
	for label, row in zip(rowlabels, m):
		print(fmt % ((label,) + tuple(row)))


def print_attrs_by_type(attrs_by_type):
	for typename, attrs in attrs_by_type.items():
		attrs = [attr for attr in attrs if not attr.startswith("_")]
		if len(attrs) > 8:
			attrs = attrs[:8] + ["..."]
		print("  %-40s %s" % (typename, " ".join(attrs)))
