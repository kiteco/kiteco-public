from typing import Any, Dict, NamedTuple

from enum import EnumMeta

import json


def serialize_config(conf: NamedTuple) -> str:
    return serialize_namedtuple(conf, pretty=True, print_enum = True)


def serialize_namedtuple(nt: NamedTuple, pretty: bool = False, print_enum: bool = False) -> str:
    def preprocess(obj: Any) -> Any:
        if hasattr(obj, '_asdict'):
            # this is most likely a NamedTuple
            return {k: preprocess(v) for k, v in obj._asdict().items()}
        elif isinstance(obj, list):
            return [preprocess(v) for v in obj]
        elif isinstance(obj, dict):
            return {k: preprocess(v) for k, v in obj.items()}
        elif isinstance(type(obj), EnumMeta):
            # obj is an enum.Enum
            if print_enum:
                return "{}.{}".format(type(obj).__name__, obj.name)
            else:
                return obj.value
        return obj

    d = preprocess(nt)

    if pretty:
        return json.dumps(d,
                          sort_keys=True,
                          indent=4,
                          separators=(',', ': '))
    else:
        return json.dumps(d)
