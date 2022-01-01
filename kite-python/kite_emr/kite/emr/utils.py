import yaml
import collections

# Ordered loading of dictionary items in yaml files
# Taken from: SO link: /questions/5121931/in-python-how-can-you-load-yaml-mappings-as-ordereddicts

def yaml_ordered_load(fp):
    class OrderedLoader(yaml.Loader):
        pass

    def construct_mapping(loader, node):
        loader.flatten_mapping(node)
        return collections.OrderedDict(loader.construct_pairs(node))

    OrderedLoader.add_constructor(
        yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
        construct_mapping)

    return yaml.load(fp, OrderedLoader)
