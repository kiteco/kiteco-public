import math
import json
import argparse
import collections
import numpy as np

from kite.typelearning import typelearning


class FunctionRecord(object):
	"""
	Represents a function together with a probability distribution over its return types
	"""
	def __init__(self, function, logp_by_type):
		self.function = function
		self.logp_by_type = logp_by_type

	def to_json(self):
		return {
			"function": self.function,
			"return_type": [
				{
					"name": typename,
					"log_probability": logp
				}
				for typename, logp in self.logp_by_type.items()
			]
		}


class TypeRecord(object):
	"""
	Represents a type together with a probability distribution over its attributes
	"""
	def __init__(self, typename, logp_by_attribute):
		self.typename = typename
		self.logp_by_attribute = logp_by_attribute

	def to_json(self):
		return {
			"type": self.typename,
			"attributes": [
				{
					"name": attr,
					"log_probability": logp
				}
				for attr, logp in self.logp_by_attribute.items()
			]
		}


class AliasRecord(object):
	"""
	Represents a fully qualified name together with an alias for that name
	"""
	def __init__(self, alias, canonical):
		self.alias = alias
		self.canonical = canonical

	def to_json(self):
		return {
			"alias": self.alias,
			"canonical": self.canonical
		}


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument("--importtree", required=True)
	parser.add_argument("--usages", required=True)
	parser.add_argument("--packages", nargs="+")
	parser.add_argument("--output_funcs", required=True)
	parser.add_argument("--output_types", required=True)
	parser.add_argument("--output_aliases", required=True)
	args = parser.parse_args()

	# Setup
	np.seterr(all="raise")

	# Load import tree
	attrs_by_type, all_funcs, all_func_aliases = typelearning.load_import_tree(args.importtree, args.packages)
	all_types = list(attrs_by_type.keys())

	# Load usages
	aliased_usages_by_func = typelearning.load_usages(args.usages, args.packages)

	# Resolve aliasing in usages
	all_usages_by_func = {}
	for func, usages in aliased_usages_by_func.items():
		normalized = all_func_aliases.get(func, None)
		if normalized is not None:
			all_usages_by_func[normalized] = usages

	# Find the intersection between functions from import analysis and function from usage counting
	funcs = sorted(set(all_funcs).intersection(all_usages_by_func.keys()))
	usages_by_func = {f: all_usages_by_func[f] for f in funcs}

	# Run training
	params = typelearning.train(attrs_by_type, usages_by_func, num_iters=1)

	# Construct the final alias records
	alias_records = [AliasRecord(alias, canonical) for alias, canonical in all_func_aliases.items()]

	# Construct the final function records
	func_records = []
	for func, row in zip(params.funcs, params.func_table):
		type_distr = {params.types[i]: math.log(row[i]) for i in np.argsort(row)[-10:]}
		func_records.append(FunctionRecord(func, type_distr))

	# Construct the final type records
	type_records = []
	for typename, col in zip(params.types, params.attr_table.T):
		attr_distr = {params.attrs[i]: math.log(col[i]) for i in np.argsort(col)[-10:]}
		type_records.append(TypeRecord(typename, attr_distr))

	# Sort by name
	alias_records.sort(key=lambda r: r.alias)
	func_records.sort(key=lambda r: r.function)
	type_records.sort(key=lambda r: r.typename)

	# Report results
	print("\nFunctions:")
	for func in usages_by_func.keys():
		print("  "+func)

	print("\nTypes:")
	typelearning.print_attrs_by_type(attrs_by_type)

	print("\nTypes -> Attributes:")
	for record in type_records:
		print("  %-40s %s" % (record.typename, typelearning.logdistr_str(record.logp_by_attribute)))

	print("\nFunctions -> Types:")
	for record in func_records:
		print("  %-40s %s" % (record.function, typelearning.logdistr_str(record.logp_by_type)))

	# Write the alias table
	with open(args.output_aliases, "w") as f:
		for record in alias_records:
			json.dump(record.to_json(), f)

	# Write the func-type table
	with open(args.output_funcs, "w") as f:
		for record in func_records:
			json.dump(record.to_json(), f)

	# Write the type-attribute table
	with open(args.output_types, "w") as f:
		for record in type_records:
			json.dump(record.to_json(), f)


if __name__ == "__main__":
	main()
