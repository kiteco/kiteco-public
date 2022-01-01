# -*- coding: utf8 -*-

import re
import sys
import math
import itertools
import collections
import copy
import heapq
import random

import numpy as np
import pandas as pd
import scipy.sparse
import sklearn.cluster

import kite.canonicalization.utils as utils

WILDCARD_SYMBOL = '*'

BLANK_SYMBOL = ' '  # a unicode char used to indicate non-matched chars

class WildcardType(object):
	def __str__(self):
		return '*'
	def __repr__(self):
		return '*'

WILDCARD = WildcardType()  # an opaque token used to indicate wildcard


class StringDiff(object):
	"""Represents a word-by-word correspondence between two strings."""
	def __init__(self, left, right, pairs, exact_pairs, leftmap, rightmap, score):
		self.left = left
		self.right = right
		self.pairs = pairs
		self.exact_pairs = exact_pairs
		self.leftmap = leftmap
		self.rightmap = rightmap
		self.score = score

	def identical(self):
		"""Return true if the left and right vectors are identical."""
		return len(self.left) == len(self.exact_pairs) == len(right)

	def compatible(self):
		"""Return true if the left and right vectors are compatible given wildcards."""
		return -1 not in self.leftmap and -1 not in self.rightmap

	def make_template(self):
		"""Construct a template that matches both of the inputs to this diff."""
		template = []
		prevpair = (-1, -1)
		for pair in self.exact_pairs + [(len(self.left), len(self.right))]:
			if prevpair[0] < pair[0]-1 or prevpair[1] < pair[1]-1:
				template.append(WILDCARD)
			if pair[0] < len(self.left):
				template.append(self.left[pair[0]])
			prevpair = pair
		return template

	def align(self):
		top = []
		bottom = []
		ai, bi = -1, -1
		for aj, bj in self.pairs + [(len(self.left), len(self.right))]:
			top += self.left[ai+1:aj] + [None]*(bj-bi-1)
			bottom += [None]*(aj-ai-1) + self.right[bi+1:bj]
			ai, bi = aj, bj
			if aj < len(self.left) and bj < len(self.right):
				top.append(self.left[aj])
				bottom.append(self.right[bj])
		return top, bottom

	def __str__(self):
		pairs = list(map(list, zip(*self.align())))
		for pair in pairs:
			if pair[0] == WILDCARD and pair[1] == WILDCARD:
				pair[0] = WILDCARD_SYMBOL
				pair[1] = WILDCARD_SYMBOL
			elif pair[0] == WILDCARD and pair[1] is None:
				pair[0] = WILDCARD_SYMBOL
				pair[1] = BLANK_SYMBOL
			elif pair[0] is None and pair[1] == WILDCARD:
				pair[0] = BLANK_SYMBOL
				pair[1] = WILDCARD_SYMBOL
			elif pair[0] == WILDCARD:
				pair[0] = WILDCARD_SYMBOL * len(pair[1])
			elif pair[1] == WILDCARD:
				pair[1] = WILDCARD_SYMBOL * len(pair[0])
			if pair[0] is None:
				pair[0] = BLANK_SYMBOL * len(pair[1])
			if pair[1] is None:
				pair[1] = BLANK_SYMBOL * len(pair[0])
		a, b = list(zip(*pairs))
		return '|%s|\n|%s|' % (' '.join(a), ' '.join(b))


def template_string(template):
	return ' '.join(WILDCARD_SYMBOL if token == WILDCARD else token for token in template)


def format_string_from_template(template):
	return ' '.join('%s' if token == WILDCARD else token for token in template)


def match(a, b):
	"""Compute a matching between two strings, either of which could contain wildcards.
	This matching can be used to calculate the following:
	  - the minimum edit distance between the strings
	  - the minimal "super template" that matches both strings
	  - the parts of one string that are captured by wildcards in the other.
	The matching is computed by a dynamic program that minimizes a weighted 
	edit distance cost."""
	EXACT_MATCH_SCORE = 20  # score for matching two identical tokens
	WHITESPACE_MATCH_SCORE = 2  # score for matching two whitespace tokens
	WILDCARD_MATCH_SCORE = 1  # score for matching a token to a wildcard
	EXACT_MATCH, LEFT_WILDCARDED, RIGHT_WILDCARDED, LEFT_OPHAN, RIGHT_ORPHAN, WILDCARD_TO_WILDCARD \
		= 1, 2, 3, 4, 5, 6

	assert a is not None
	assert b is not None
	assert len(a) <= 50  # may reach maximum recursion limit for very long vectors
	assert len(b) <= 50  # may reach maximum recursion limit for very long vectors
	assert WILDCARD_MATCH_SCORE < WHITESPACE_MATCH_SCORE < EXACT_MATCH_SCORE

	cache = {}

	def solve(i, j):
		"""The solver that looks up cached solutions when possible."""
		solution = cache.get((i, j), None)
		if solution is None:
			solution = solve_impl(i, j)
			cache[(i, j)] = solution
		return solution

	def solve_impl(i, j):
		"""The core subproblem solver."""
		# Base case: at end of string
		if i >= len(a) or j >= len(b):
			return (0, None)

		candidates = []

		# Recursive case 1: orphan the left token
		score, _ = solve(i+1, j)
		if b[j] == WILDCARD:
			candidates.append((score+WILDCARD_MATCH_SCORE, LEFT_WILDCARDED))
		else:
			candidates.append((score, LEFT_OPHAN))

		# Recursive case 2: orphan the right token
		score, _ = solve(i, j+1)
		if a[i] == WILDCARD:
			candidates.append((score+WILDCARD_MATCH_SCORE, RIGHT_WILDCARDED))
		else:
			candidates.append((score, RIGHT_ORPHAN))

		# Recursive case 3: exact match
		matchscore = 0
		if a[i] == b[j]:
			score, _ = solve(i+1, j+1)
			if a[i] == WILDCARD and b[j] == WILDCARD:
				candidates.append((score, WILDCARD_TO_WILDCARD))
			elif a[i].isspace() and b[j].isspace():
				candidates.append((score + WHITESPACE_MATCH_SCORE, EXACT_MATCH))
			else:
				candidates.append((score + EXACT_MATCH_SCORE, EXACT_MATCH))

		# Find maximum
		return max(candidates, key=lambda x: x[0])

	# Run the dynamic program
	try:
		total_score, _ = solve(0, 0)
	except Exception as ex:
		print('Failed to match:')
		print('  ', a)
		print('  ', b)
		raise ex

	# Backtrack
	pairs = []
	exact_pairs = []
	leftmap = [None] * len(a)
	rightmap = [None] * len(b)
	i = j = 0
	while True:
		_, pointer = cache[(i, j)]
		if pointer is None:
			break
		if pointer == EXACT_MATCH:
			exact_pairs.append((i, j))
			pairs.append((i, j))
			leftmap[i] = j
			rightmap[j] = i
			i += 1
			j += 1
		elif pointer == LEFT_OPHAN:
			i += 1
		elif pointer == RIGHT_ORPHAN:
			j += 1
		elif pointer == LEFT_WILDCARDED:
			leftmap[i] = j
			pairs.append((i, j))
			i += 1
		elif pointer == RIGHT_WILDCARDED:
			rightmap[j] = i
			pairs.append((i, j))
			j += 1
		else:
			raise Exception('Invalid pointer: '+str(pointer))

	# Return the final diff
	return StringDiff(a, b, pairs, exact_pairs, leftmap, rightmap, total_score)


def is_compatible(tokens, template):
	"""Determine whether a leaf is compatible with a given template."""
	indices = [0]
	for token in tokens:
		assert token != WILDCARD, "is_compatible found wildcard in leaf"
		next_indices = []
		for i in indices:
			if i < len(template):
				if template[i] == token:
					next_indices.append(i+1)
				elif template[i] == WILDCARD:
					next_indices.append(i)
					next_indices.append(i+1)
		if not next_indices:
			return False
		indices = next_indices
	return len(template) in indices


def tokenize(s):
	"""A simple state machine to tokenize error messages."""
	SPLIT_CHARS = ':()'
	prevsplit = True
	begin = 0
	tokens = []

	for i, c in enumerate(s):
		splitter = c.isspace() or c in SPLIT_CHARS
		if prevsplit:
			if i != begin and not s[begin:i].isspace():
				tokens.append(s[begin:i])
			begin = i
			if not splitter:
				prevsplit = False
		else:
			if splitter:
				if i != begin:
					tokens.append(s[begin:i])
					begin = i
				prevsplit = True

	if begin != len(s):
		tokens.append(s[begin:len(s)])

	return tokens


def label_connected_components(num_nodes, edges):
	"""Given a graph described by a list of undirected edges, find all connected
	components and return labels for each node indicating which component they belong to."""
	leader = list(range(num_nodes))

	def head(k):
		if leader[k] == k:
			return k
		else:
			leader[k] = head(leader[k])
			return leader[k]

	for i, j in edges:
		hi, hj = head(i), head(j)
		if hi != hj:
			leader[hi] = hj

	leaders = [head(i) for i in range(num_nodes)]
	reduction = {leader: index for index, leader in enumerate(set(leaders))}
	return [reduction[leader] for leader in leaders]


class Candidate(object):
	"""Represents a candidate for a pair of templates to merge in the agglomerative
	clustering algorith."""
	def __init__(self, i, j, diff, score):
		self.i = i
		self.j = j
		self.diff = diff
		self.score = score
	def __lt__(self, rhs):
		# Use > here so that we get the highest score first
		return self.score > rhs.score
	def __le__(self, rhs):
		# Use >= here so that we get the highest score first
		return self.score >= rhs.score


def compute_edit_distance(a, b):
	diff = match(a, b)
	return float(len(a) + len(b) - 2*len(diff.exact_pairs)) / float(len(a) + len(b))


def compute_matching_score(a, b):
	diff = match(a, b)
	score = float(len(diff.exact_pairs)) / max(len(a), len(b))
	return diff, score


def discover_templates(tokenvecs, min_members=5, algorithm='flat_agglomerative'):
	"""Given a list of tokenized error messages, find a set of templates that best explain
	the error messages and return all templates matching at least MIN_MEMBERS of the
	error messages."""
	# Set of items currently in the index
	indexed_items = set()

	# Map from token to errors containing that token
	inverted_index = collections.defaultdict(list)

	def compute_idf(word):
		index_bin = inverted_index[word] 
		if len(index_bin) == 0:
			return idf_normalizer  # corresponds to log(1/N)
		else:
			return idf_normalizer - math.log(len(inverted_index[word]))

	def add_to_index(tokenvec, i):
		indexed_items.add(i)
		for word in set(tokenvec):
			if word != WILDCARD:
				inverted_index[word].append(i)

	def remove_from_index(i):
		indexed_items.remove(i)

	def find_neighbors(tokenvec, n, cutoff=1000):
		assert tokenvec is not None
		all_indices = set()
		scores = [0] * (max(indexed_items) + 1)
		for word in tokenvec:
			idf = compute_idf(word)
			index_bin = inverted_index[word]
			if len(index_bin) < cutoff:   # huge bins are useless
				for j in index_bin:
					if j in indexed_items:
						scores[j] += idf
						all_indices.add(j)

		return heapq.nlargest(n, all_indices, key=lambda i: scores[i])

	def template_cost(template):
		"""Compute the cost for a template."""
		cost = 0.
		for token in template:
			if token == WILDCARD:
				cost += WILDCARD_COST
			else:
				cost += compute_idf(token)
		return cost

	def leaf_cost(leaf, template):
		"""Compute the cost for a leaf with an associated template."""
		diff = match(leaf, template)

		# Can only diff when left is compatible with template
		assert diff.compatible(), "not compatible: '%s' and '%s'" % (diff.left, diff.right)

		# Find the number of unexplained words, which are the words matched with wildcards
		cost = 0.
		for word, counterpart in zip(leaf, diff.leftmap):
			if template[counterpart] == WILDCARD:
				cost += compute_idf(word)
		return cost

	# Construct an inverted index to try to discover candidate pairs
	print('Building inverted index...')
	for i, vec in enumerate(tokenvecs):
		add_to_index(vec, i)

	idf_normalizer = math.log(sum(len(x) for x in inverted_index.values()))


	########################################
	# SPECTRAL CLUSTERING
	########################################
	if algorithm == 'spectral':
		# Number of clusters to compute during spectral clustering
		NUM_CLUSTERS = 200
		NUM_RANDOM_LINKS = 1
		NUM_NEIGHBORS = 250
		RBF_GAMMA = -3.

		# Construct an affinity matrix
		print('Computing affinity map...')
		affinitymap = {}
		for i in range(len(tokenvecs)):
			affinitymap[(i, i)] = 1.

			if (i+1) % 1000 == 0:
				print('  Processing element %d of %d' % (i+1, len(tokenvecs)))
			# Add distances to some random points
			nearest_neighbors = list(find_neighbors(tokenvecs[i], NUM_NEIGHBORS))
			random_neighbors = [random.randint(0, len(tokenvecs)-1) for _ in range(NUM_RANDOM_LINKS)]
			for j in set(nearest_neighbors + random_neighbors):
				if i != j:
					dist = compute_edit_distance(tokenvecs[i], tokenvecs[j])
					af = math.exp(RBF_GAMMA * dist*dist)
					affinitymap[(i, j)] = af
					affinitymap[(j, i)] = af

		# Construct sparse matrix
		edges, affinityvec = list(zip(*affinitymap.items()))
		rows, cols = list(zip(*edges))
		affinity = scipy.sparse.csr_matrix((affinityvec, (rows, cols)))

		# Divide into connected components
		component_labels = label_connected_components(len(tokenvecs), edges)
		num_components = max(component_labels)+1
		print('Found %d connected components' % num_components)

		components = [[] for _ in range(num_components)]
		for i, label in enumerate(component_labels):
			components[label].append(i)

		# Do spectral clustering
		if len(components) == 1:
			cl = sklearn.cluster.SpectralClustering(
				n_clusters=NUM_CLUSTERS,
				affinity='precomputed',
				eigen_solver='amg')
			
			labels = cl.fit_predict(affinity)

		else:
			next_label = 0
			labels = [None for _ in range(len(tokenvecs))]
			for i, component in enumerate(components):
				print('Running spectral clustering on component %d of %d...' % (i+1, len(components)))
				print('  size:', len(component))

				local_num_clusters = min(len(component)-1, NUM_CLUSTERS * len(component) / len(tokenvecs))
				print('  local num clusters:', local_num_clusters)

				if len(component) == 1 or local_num_clusters <= 1:
					print('   degenerate')
					local_labels = [0] * len(component)

				else:
					cl = sklearn.cluster.SpectralClustering(
						n_clusters=local_num_clusters,
						affinity='precomputed',
						eigen_solver='amg')
					
					local_labels = cl.fit_predict(affinity[component, component])

				# Propagate labels to global list
				for idx, label in zip(component, local_labels):
					labels[idx] = next_label + label

				next_label += len(set(local_labels))

		# Assign tokens to labels
		num_labels = len(set(labels))
		tokenvecs_by_label = [[] for _ in range(num_labels)]
		for tokenvec, label in zip(tokenvecs, labels):
			print('label:', label, ' num labels:', num_labels)
			tokenvecs_by_label[label].append(tokenvec)

		# Compute a template for each label
		templates = []
		for label, vecs in enumerate(tokenvecs_by_label):
			if len(vecs) == 0:
				template = []
			else:
				template = vecs[0]
				for tokens in vecs[1:]:
					if not is_compatible(tokens, template):
						template = match(template, tokens).make_template()
			templates.append(template)

	########################################
	# AGGLOMERATIVE CLUSTERING
	########################################
	elif algorithm == 'flat_agglomerative':
		# The total cost of an ontology is:
		#   sum(IDF for each word) + WILDCARD_COST * nwildcards
		# where nwildcards is the total number of wildcards in the ontology.
		WILDCARD_COST = 4

		# Number of candidates to insert for each new templates
		COMPARISONS_PER_LINE = 100

		# Initialize all templates
		templates = copy.deepcopy(tokenvecs)
		labels = list(range(len(tokenvecs)))

		def find_candidates_for(tokenvec, n):
			# Sort the candidate matches by num matching words
			for j in find_neighbors(tokenvec, n):
				if templates[j] is not None:
					diff, score = compute_matching_score(tokenvec, templates[j])
					yield (j, diff, score)

		# Compute matching distance between N pairs
		print('Computing initial merge candidates...')
		candidates = []
		for i, wordvec in enumerate(templates):
			for j, diff, score in find_candidates_for(templates[i], COMPARISONS_PER_LINE):
				if j != i:
					candidates.append(Candidate(i, j, diff, score))

		# Make into a heap
		heapq.heapify(candidates)
		rejected_indices = set()
		rejected_templates = set()

		# Agglomerate
		print('Beginning agglomeration...')
		while len(candidates) > 0:
			# Pop the top element
			c = heapq.heappop(candidates)
			assert c.i != c.j

			# These templates may have already been merged
			if templates[c.i] is None or templates[c.j] is None:
				continue

			if (min(c.i, c.j), max(c.i, c.j)) in rejected_indices:
				continue

			print('\nConsidering match %d -> %d (score=%f, %d in queue):' % (i, j, score, len(candidates)))
			print('    ' + template_string(templates[c.i]))
			print('    ' + template_string(templates[c.j]))

			# Find the smallest template that matches both templates
			template_diff = match(templates[c.i], templates[c.j])
			super_template = template_diff.make_template()

			if template_string(super_template) in rejected_templates:
				rejected_indices.add((min(c.i, c.j), max(c.i, c.j)))
				print('  Template in rejected set')
				continue

			# Find templates that would be deleted if we adopt the proposed leaves
			compatible_leaves = []
			new_leaf_counts = [0] * len(templates)
			for i, leaf in enumerate(tokenvecs):
				if is_compatible(leaf, super_template):
					compatible_leaves.append(i)
				else:
					new_leaf_counts[labels[i]] += 1

			unneeded_templates = []
			for i, count in enumerate(new_leaf_counts):
				if count == 0 and templates[i] is not None:
					unneeded_templates.append(i)

			# Compute the cost for the current configuration
			before_cost_t = sum(template_cost(templates[i]) for i in unneeded_templates)
			before_cost_l = sum(leaf_cost(tokenvecs[i], templates[labels[i]]) for i in compatible_leaves)
			before_cost = before_cost_t + before_cost_l

			# Compute cost for the merged template
			after_cost_t = template_cost(super_template)
			after_cost_l = sum(leaf_cost(tokenvecs[i], super_template) for i in compatible_leaves)
			after_cost = after_cost_t + after_cost_l

			print('  Super template:', template_string(super_template))
			print('  Would adopt %d leaves and displace %d templates' % (len(compatible_leaves), len(unneeded_templates)))
			print('  Cost before: %.1f (%.1f leaf + %.1f template)' % (before_cost, before_cost_l, before_cost_t))
			print('  Cost after: %.1f (%.1f leaf + %.1f template)' % (after_cost, after_cost_l, after_cost_t))

			if after_cost < before_cost:
				print('  ## Accepting!')
				# If the merged cost is smaller than the current cost then do the merge
				new_index = len(templates)
				templates.append(super_template)
				for i in unneeded_templates:
					print('    deleting template: ', template_string(templates[i]))
					templates[i] = None
					remove_from_index(i)
				for i in compatible_leaves:
					labels[i] = new_index

				# Generate new candidates for this new template
				add_to_index(super_template, new_index)
				for j, diff, score in find_candidates_for(super_template, COMPARISONS_PER_LINE):
					if j != new_index and templates[j] is not None:
						heapq.heappush(candidates, Candidate(i, j, diff, score))

			else:
				rejected_indices.add((min(c.i, c.j), max(c.i, c.j)))
				rejected_templates.add(template_string(super_template))

	else:
		raise Exception('Invalid algorithm: "%s"' % algorithm)

	# Print the final templates with the errors that matched to each
	print('\nFinal templates:\n')
	for label, template in enumerate(templates):
		if template is not None:
			print(template_string(template))
			for i, tokens in enumerate(tokenvecs):
				if labels[i] == label:
					print('  ' + ' '.join(map(str, tokens)))

	# Count the members
	members = [[] for _ in templates]
	for label, tokenvec in zip(labels, tokenvecs):
		members[label].append(tokenvec)

	# Renumber templates
	new_members = []
	new_templates = []
	for template, member in zip(templates, members):
		if template is not None and len(member) >= min_members:
			new_templates.append(template)
			new_members.append(member)

	return new_templates, new_members
