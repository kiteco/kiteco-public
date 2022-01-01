from typing import NamedTuple

from .attr_base import Request as AttrBaseRequest

from ..infer_call.request import ArgTypeRequest, KwargRequest, ArgPlaceholderRequest, Request as CallRequest
from ..infer_attr.request import Request as AttrRequest


class Request(NamedTuple):
    max_samples: int
    call: CallRequest
    attr: AttrRequest
    attr_base: AttrBaseRequest
    arg_type: ArgTypeRequest
    kwarg_name: KwargRequest
    arg_placeholder: ArgPlaceholderRequest

    def to_json(self) -> dict:
        return {
            'max_samples': self.max_samples,
            'call': self.call.to_json(),
            'attr': self.attr.to_json(),
            'attr_base': self.attr_base.to_json(),
            'arg_type': self.arg_type.to_json(),
            'kwarg_name': self.kwarg_name.to_json(),
            'arg_placeholder': self.arg_placeholder.to_json(),
        }
