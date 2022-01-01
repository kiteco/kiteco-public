"""
Functions for reading the Instrumental YAML spec
"""

import yaml

from instrumental import Dashboard, Graph, Metric

def read(filename="dashboards.yaml"):
    """Reads dashboard data from YAML file into its respective classes

    See the README for the expected structure of the YAML file, as well as the `validate` function
    """
    with open(filename) as f:
        data = yaml.load(f)

    # read templates
    graph_templates, dashboard_templates = {}, {}

    if "graph_templates" in data:
        for k, v in data["graph_templates"].items():
            # for templates, the title is specified as a field rather than the key name
            title = v.pop("title")
            # for reading templates, the constructor functions should only return one item
            graph_templates[k] = read_graph(title, v)[0]

    if "dashboard_templates" in data:
        for k, v in data["dashboard_templates"].items():
            name = v.pop("name")
            dashboard_templates[k] = read_dashboard(
                name, v, graph_templates, dashboard_templates)[0]

    # read dashboards
    dashboards = []
    for dashboard in data["dashboards"]:
        for name, values in dashboard.items():
            dashboards.extend(read_dashboard(name, values, graph_templates, dashboard_templates))

    return dashboards

def read_graph(title, data, templates=None):
    """Construct graphs from the given data"""

    # if title starts with t_, copy from template and return
    if title.startswith("t_"):
        key = title.split("t_").pop()
        return templates[key].copy(**data)

    # read metrics
    metrics = []
    for metrics_data in data["metrics"]:
        # if item is a dict, call read_metrics
        if isinstance(metrics_data, dict):
            metrics.extend(read_metric(metrics_data))
        # otherwise, just add (should be just a string)
        else:
            metrics.extend(Metric.new(metrics_data))

    # delete metrics items since we don't want to pass it in
    del data["metrics"]

    # create graph(s)
    graphs = Graph.new(title, metrics, **data)
    return graphs

def read_metric(metrics_data):
    """Construct metrics from the given data"""
    # metrics_data should be a dict that looks like {<metrics string>: <dict of copy values>}
    metrics = []
    for metric, values in metrics_data.items():
        metrics.extend(Metric.new(metric, **values))

    return metrics

def read_dashboard(name, data, graph_templates=None, dashboard_templates=None):
    """Construct a Dashboard from the given name and data"""

    # if name starts with t_, copy from template and return
    if name.startswith("t_"):
        key = name.split("t_").pop()
        return dashboard_templates[key].copy(**data)

    # read graphs
    graphs = []
    for graph in data["graphs"]:
        # if item is a dict, call read_graphs
        if isinstance(graph, dict):
            for title, graph_data in graph.items():
                graphs.extend(read_graph(title, graph_data, graph_templates))

        # if item is just a string, should be a template name so get the key and copy
        # NOTE: this should only in a dashboard template
        else:
            key = graph.split("t_").pop()
            graphs.extend(graph_templates[key].copy())

    # delete metrics items since we don't want to pass it in
    del data["graphs"]

    # create dashboard(s)
    dashboards = Dashboard.new(name, graphs=graphs, **data)
    return dashboards

def append_or_extend(dest, items):
    """Append or extend"""
    if isinstance(items, list):
        dest.extend(items)
    else:
        dest.append(items)
