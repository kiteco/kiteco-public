import unittest

from . import dynamic


class Foo(object):
	"""A mock class"""
	def bar(self):
		"""Does nothing"""
		pass


class TypeutilsTest(unittest.TestCase):
	def test_qualname(self):
		f = Foo()
		self.assertEqual(dynamic.qualname(f.bar), "Foo.bar")
		self.assertEqual(dynamic.qualname(Foo.bar), "Foo.bar")

	def test_classify(self):
		f = Foo()
		self.assertEqual(dynamic.classify(f), "object")
		self.assertEqual(dynamic.classify(Foo), "type")
		self.assertEqual(dynamic.classify(dynamic), "module")
		self.assertEqual(dynamic.classify(Foo.bar), "function")
		self.assertEqual(dynamic.classify(f.bar), "function")

	def test_fullname(self):
		f = Foo()
		self.assertEqual(dynamic.fullname(unittest), "unittest")
		self.assertEqual(dynamic.fullname(unittest.TestCase), "unittest.case.TestCase")
		self.assertEqual(dynamic.fullname(unittest.TestCase.assertEqual), "unittest.case.TestCase.assertEqual")
		self.assertEqual(dynamic.fullname(1), None)

	def test_package(self):
		f = Foo()
		self.assertEqual(dynamic.package(unittest), "unittest")
		self.assertEqual(dynamic.package(unittest.TestCase), "unittest.case")
		self.assertEqual(dynamic.package(unittest.TestCase.assertEqual), "unittest.case")
		self.assertEqual(dynamic.package(1), None)

	def test_doc(self):
		self.assertEqual(dynamic.doc(Foo), "A mock class")
		self.assertEqual(dynamic.doc(Foo.bar), "Does nothing")
