import json


def loadjson(f, bufsize=1000000, **kwargs):
	"""
	Load a sequence of json objects from a stream
	"""
	decoder = json.JSONDecoder(**kwargs)
	last_error = None
	cur = ""
	while True:
		buf = f.read(bufsize)
		if buf:
			cur += buf.decode()
			cur = cur.lstrip() # in rare cases, there are spaces in front causing problems decoding
		elif last_error is not None:
			raise last_error
		else:
			return

		while cur:
			# Consume an object if possible
			try:
				last_error = None
				obj, consumed = decoder.raw_decode(cur)
				cur = cur[consumed:]
			except ValueError as ex:
				last_error = ex
				break

			# consume whitespace, if any
			offset = 0
			while offset < len(cur) and cur[offset].isspace():
				offset += 1
			cur = cur[offset:]

			# yield the object that was consumed
			yield obj


