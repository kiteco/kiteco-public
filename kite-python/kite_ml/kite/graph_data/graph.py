from typing import NewType

from enum import Enum

NodeID = NewType('NodeID', int)
VariableID = NewType('VariableID', int)


class NodeType(Enum):
    AST_INTERNAL_NODE = 'ast_internal_node'
    AST_TERMINAL_NODE = 'ast_terminal_node'
    VARIABLE_USAGE_NODE = 'variable_usage_node'


class EdgeType(Enum):
    AST_CHILD = 'ast_child'
    NEXT_TOKEN = 'next_token'
    LAST_LEXICAL_USE = 'last_lexical_use'
    COMPUTED_FROM = 'computed_from'
    LAST_READ = 'last_read'
    LAST_WRITE = 'last_write'
    DATA_FLOW = 'data_flow'
    AST_CHILD_ATTR_VALUE = 'ast_child_attr_value'
    AST_CHILD_ARG_VALUE = 'ast_child_arg_value'
    AST_CHILD_ASSIGN_RHS = 'ast_child_assign_rhs'
    RETURN_VALUE_OF = 'return_value_of'
    SCOPE = 'scope'

    @staticmethod
    def from_edge_key(k: str) -> 'EdgeType':
        assert k.endswith('_forward') or k.endswith('_backward'), \
            'expected edge key {0} to end with _forward or _backward'.format(k)
        if k.endswith('_forward'):
            return EdgeType(k[:k.index('_forward')])
        return EdgeType(k[:k.index('_backward')])

    @staticmethod
    def assert_edge_key_valid(k: str):
        assert k.endswith('_forward') or k.endswith('_backward'), \
            'expected edge key {0} to end with _forward or _backward'.format(k)
        if k.endswith('_forward'):
            typ = k[:k.index('_forward')]
        else:
            typ = k[:k.index('_backward')]
        EdgeType(typ)
        return None

    def edge_key(self, forward: bool) -> str:
        if forward:
            return self.value + '_forward'
        return self.value + '_backward'

    @staticmethod
    def reversed_edge_key(k: str) -> str:
        assert k.endswith('_forward') or k.endswith('_backward'), \
            'expected edge key {0} to end with _forward or _backward'.format(k)

        if k.endswith('_forward'):
            return k[:k.index('_forward')] + '_backward'
        return k[:k.index('_backward')] + '_forward'
