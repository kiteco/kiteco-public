import logging
import json
from .explore import Explorer


def explore_and_link(package, runtime_refmap):
    result = Explorer.explore_package(package, refmap=runtime_refmap)
    nodes_by_id = result['info_by_id']

    linked_nodes = {}
    output = []
    def link_node(node_id):
        if node_id not in nodes_by_id:
            return None
        if node_id in linked_nodes:
            return linked_nodes[node_id]

        node = nodes_by_id[node_id]
        linked_node = {'@node': node}
        linked_nodes[node_id] = linked_node

        if 'members' in node:
            linked_members = {attr: link_node(ref) for attr, ref in node['members'].items()}
            linked_node.update(linked_members)

        if 'type_id' in node:
            linked_node['@type'] = link_node(node['type_id'])

        # find the root node
        if node['canonical_name'] == package:
            output.append(linked_node)

        return linked_node

    list(map(link_node, nodes_by_id))

    print(output)
    assert len(output) == 1
    return output[0]


def test_explore_example_references(runtime_refmap):
    logging.basicConfig(level=logging.DEBUG)
    graph = explore_and_link('kite.pkgexploration.examples.explore', runtime_refmap)

    assert graph['Test']['large']['@node']['reference'] == 'kite.pkgexploration.examples.runtime_helper.LARGE'
    assert graph['Test']['foo']['@node']['reference'] == 'kite.pkgexploration.examples.foo.Foo.foo'
    assert graph['Foo']['@node']['reference'] == 'kite.pkgexploration.examples.foo.Foo'
    assert graph['foo']['@type']['@node']['reference'] == 'kite.pkgexploration.examples.foo.Foo'
