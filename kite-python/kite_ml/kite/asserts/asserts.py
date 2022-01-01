from enum import Enum, EnumMeta

from typing import Any, Callable, List, Optional, Sized

# An Assertion is a function that accepts a field name and its value. If the value is deemed invalid, it raises
# an AssertionError.
Assertion = Callable[[str, Any], None]

# Builder for a field
Builder = Callable[[Any], Any]


class FieldValidator(object):
    def __init__(self, clz: type, d: dict):
        self.namespace = clz.__name__
        assert isinstance(d, dict), '{0}: expected dict, got {1}'.format(self.namespace, type(d))
        self.d = d

    def get(self,
            field_name: str,
            instance_of: type,
            asserts: Optional[Assertion]=None,
            build: Optional[Builder]=None):

        assert isinstance(field_name, str)
        assert isinstance(instance_of, type)

        assert field_name in self.d, 'missing {}'.format(self._fqn(field_name))
        val = self.d[field_name]

        assert isinstance(val, instance_of), \
            '{0}: expected type {1} but got type {2}'.format(self._fqn(field_name), instance_of, type(val))

        if asserts is not None:
            asserts(self._fqn(field_name), val)

        if build is not None:
            return build(val)
        return val

    def get_float(self, field_name: str, asserts: Optional[Assertion]=None) -> float:
        val = self.get(field_name, object, asserts=asserts)
        assert isinstance(val, int) or isinstance(val, float), \
            '{0}: expected int/float but got type {1}'.format(self._fqn(field_name), type(val))
        return float(val)

    def get_enum(self, field_name: str, enum_class: EnumMeta):
        return enum_class(self.get(field_name, object, build=enum_class))

    def get_list(self,
                 field_name: str,
                 instance_of: type,
                 build_elem: Optional[Builder] = None,
                 min_len: int=0,
                 asserts_list: Optional[Assertion]=None):
        lst = self.get(field_name, list)

        if min_len > 0:
            Assert.has_at_least_len(min_len)(self._fqn(field_name), lst)

        if asserts_list is not None:
            asserts_list(self._fqn(field_name), lst)

        ret = []

        for i, val in enumerate(lst):
            assert isinstance(val, instance_of), \
                '{0}: expected type {1} but got type {2}'.format(self._ind_fqn(field_name, i), instance_of, type(val))

            if build_elem is not None:
                val = build_elem(val)
            ret.append(val)

        return ret

    def get_map(self,
                field_name: str,
                key_instance_of: type,
                val_instance_of: type,
                val_build: Optional[Builder] = None):
        d = self.get(field_name, dict)

        ret = {}

        for key, val in d.items():
            assert isinstance(key, key_instance_of), '{0}: expected key type {1} but got type {2}'.format(
                    self._ind_fqn(field_name, key), key_instance_of, type(key))

            assert isinstance(val, val_instance_of), '{0}: expected val type {1} but got type {2}'.format(
                    self._ind_fqn(field_name, key), val_instance_of, type(val))

            if val_build is not None:
                val = val_build(val)
            ret[key] = val

        return ret

    def _fqn(self, field_name: str) -> str:
        return ".".join([self.namespace, field_name])

    def _ind_fqn(self, field_name: str, k: Any) -> str:
        return "{}.{}[{}]".format(self.namespace, field_name, k)


class Validator(object):
    def __init__(self, name: str, instance_of: type, asserts: Optional[Assertion] = None):
        assert isinstance(name, str)
        assert isinstance(instance_of, type)

        self.name = name
        self.instance_of = instance_of
        self.asserts = asserts


class Assert(object):
    @staticmethod
    def wrap(naked: Callable[[Any], None]) -> Assertion:
        def assertion(_: str, x: Any):
            naked(x)
        return assertion

    @staticmethod
    def greater_than_or_equal(to: int) -> Assertion:
        def assertion(name: str, x: Any):
            assert x >= 0, 'expected {0} to be greater than or equal to {1}, is {2}'.format(name, str(to), str(x))
        return assertion

    @staticmethod
    def one_of(elems: List[str]) -> Assertion:
        def assertion(name: str, elem: Any):
            assert elem in elems, 'expected {0} to be one of {1}, is {2}'.format(name, ' , '.join(elems), elem)
        return assertion

    @staticmethod
    def is_enum(e: EnumMeta) -> Assertion:
        def assertion(name: str, elem: Any):
            valid_elems = [v.value for v in e.__members__]
            assert elem in valid_elems, '{0} is not valid member of {1}'.format(name, e.__name__)
        return assertion

    @staticmethod
    def categorical(high: int, low: int = 0) -> Assertion:
        """
        Check that all values are in low <= val < high
        :param high:
        :param low:
        :return:
        """
        def assertion(name: str, x: Any):
            assert isinstance(x, int), 'expected type of {0} to be int, is {1}'.format(name, type(x))
            assert x >= low, 'expected {0} to be >= {1}; got {2}'.format(name, low, x)
            assert x < high, 'expected {0} to be < {1}; got {2}'.format(name, high, x)
        return assertion

    @staticmethod
    def is_type(t: type) -> Assertion:
        def assertion(name: str, x: Any):
            assert isinstance(x, t), 'expected type of {0} to be {1}, is {2}'.format(name, t, type(x))
        return assertion

    @staticmethod
    def has_len(count: int) -> Assertion:
        def assertion(name: str, x: Sized):
            assert len(x) == count, 'expected {0} to have {1} elements, has {2}'.format(name, count, len(x))
        return assertion

    @staticmethod
    def has_at_least_len(count: int) -> Assertion:
        def assertion(name: str, x: Sized):
            assert len(x) >= count, 'expected {0} to have at least {1} elements, has {2}'.format(name, count, len(x))
        return assertion

    @staticmethod
    def unique() -> Assertion:
        def assertion(name: str, l: List):
            assert len(set(l)) == len(l), \
                'expected {0} to have unique elements, has {1}'.format(name, ' , '.join([str(x) for x in l]))
        return assertion

    @staticmethod
    def chain(*args: Assertion) -> Assertion:
        def chained(name, val):
            for a in args:
                a(name, val)

        return chained

    @staticmethod
    def map(a: Assertion) -> Assertion:
        def assertion(name: str, x: List):
            assert isinstance(x, list)
            for i, el in enumerate(x):
                a("{0}_{1}".format(name, i), el)
        return assertion

    @staticmethod
    def is_2d_list_with_type(t: type) -> Assertion:
        def assertion(name: str, l: Any):
            assert isinstance(l, list), 'expected {} to have type list, got {}'.format(name, type(l))
            assert len(l) > 0, 'expected {} to have atleast len 1'.format(name)
            num = -1  # to make lint happy...
            for i, el in enumerate(l):
                assert isinstance(el, list), \
                    'expected elements of {} to be lists, got {} for {}'.format(name, type(el), i)

                if i == 0:
                    num = len(el)
                assert len(el) == num, \
                    'expected each element of {} to have length {}, got {} for {}'.format(name, num, len(el), i)

                for j, ell in enumerate(el):
                        assert isinstance(ell, t), 'expected innermost elements of {} ' \
                                                   'to have type {}, got {} for {}{}'.format(name, t, type(ell), i, j)
        return assertion

    @staticmethod
    def valid(d: dict, ns: str, validators: List[Validator]):
        assert d is not None

        for validator in validators:
            key = validator.name
            assert key in d, 'missing name {0} for {1}'.format(key, ns)

            val = d[key]

            fqn = ns + '.' + key
            assert isinstance(val, validator.instance_of), \
                'expected type {0} got type {1} for {2}'.format(str(validator.instance_of), type(val), fqn)
            if validator.asserts is not None:
                validator.asserts(fqn, val)


def assert_enum(enum_subtype: EnumMeta, inst: Any):
    assert isinstance(enum_subtype, EnumMeta)
    assert inst in list(enum_subtype), 'expected {} in {}'.format(inst, list(enum_subtype))


def assert_valid_segmented_dataset(batch_size: int, max_elem: int, elems: List[int], segment_ids: List[int]):
    assert len(elems) == len(segment_ids), \
        'num segmented elems {} != num segment ids {}'.format(len(elems), len(segment_ids))

    seen = set()
    for i in range(len(elems)):
        if max_elem > -1:
            assert 0 <= elems[i] < max_elem, 'elem {} at {} not in [0,...,{}]'.format(elems[i], i, max_elem)
        sid = segment_ids[i]
        if batch_size > -1:
            assert 0 <= sid < batch_size, 'segment id {} at  {} not in [0,...,{}]'.format(sid, i, batch_size)
        if i < len(elems)-1:
            sidn = segment_ids[i+1]
            assert sid <= sidn, \
                'segment ids must be in ascending order {} at {} is larger than {} at {}'.format(sid, i, sidn, i+1)
        seen.add(sid)

    if batch_size > -1:
        assert len(seen) == batch_size, \
            'expected atleast one sample per batch {} only got {}'.format(batch_size, len(seen))


def assert_valid_segment_ids(batch_size: int, segment_ids: List[int]):
    seen = set()
    for i in range(len(segment_ids)):
        sid = segment_ids[i]
        if batch_size > -1:
            assert 0 <= sid < batch_size, 'segment id {} at  {} not in [0,...,{}]'.format(sid, i, batch_size)
        if i < len(segment_ids)-1:
            sidn = segment_ids[i+1]
            assert sid <= sidn, \
                'segment ids must be in ascending order {} at {} is larger than {} at {}'.format(sid, i, sidn, i+1)
        seen.add(sid)

    if batch_size > -1:
        assert len(seen) == batch_size, \
            'expected atleast one sample per batch{} only got {}'.format(batch_size, len(seen))
