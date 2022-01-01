"""
Classes and functions for interacting with the Instrumental API
"""

import os
import sys
import string
import requests
import read_yaml

auth = {"X-Instrumental-Token": os.environ["INSTRUMENTAL_TOKEN"]}

def _api(method, url, data=None):
    """Instrumental API caller helper"""
    base = "https://instrumentalapp.com/api/2/organizations/{org}/projects/{proj}".format(
        org="1659", proj="2171")

    headers = {}
    headers.update(auth)

    if data is None:
        data = {}

    response = requests.request(method, base+url, headers=headers, json=data)

    return response

class Dashboard(object):
    # pylint: disable=attribute-defined-outside-init
    """Instrumental dashboard"""
    @staticmethod
    def new(name, dash_id=None, graphs=None, **values):
        """Similar to an __init__, but can use copy to return multiple objects"""
        base = Dashboard()

        base.name = string.Template(name)
        base.dash_id = dash_id
        base.graphs = graphs

        # if no values are given, return the base (this also prevents infinite copy recursion)
        if not values:
            return [base]

        # otherwise, return copy of the base
        return base.copy(**values)

    def delete(self):
        """Delete the dashboard"""
        resp = _api("DELETE", "/dashboards/{}".format(self.dash_id))
        return resp

    def create(self):
        """Create a new dashboard"""
        resp = _api("POST", "/dashboards/", data=self.to_dict()).json()
        self.dash_id = resp["id"]

        # populate dashboard IDs for graphs and create
        if self.graphs:
            for graph in self.graphs:
                graph.dash_id = self.dash_id

    def sub(self, **values):
        """Implement `sub` for `cross_product_copy`"""

        # call sub on graphs, making sure not to modify the original
        graphs = [graph.sub(**values) for graph in self.graphs]
        # NOTE: new with no values is guaranteed to return one object
        return Dashboard.new(self.name.safe_substitute(values), self.dash_id, graphs).pop()

    def copy(self, **values):
        """Copy using `cross_product_copy`, also copying all graphs and passing down values to the
        graphs"""

        # # call sub to make a new dashboard so we don't modify the original
        new_dash = self.sub()

        # copy dash
        values = clean_keys(values, self.name)
        return cross_product_copy(new_dash, **values)

    @staticmethod
    def get_all():
        """Get all dashboards"""
        dashboards = []

        resp = _api("GET", "/dashboards").json()
        for dash in resp:
            dashboards.append(Dashboard.new(name=dash["name"], dash_id=dash["id"]).pop())

        return dashboards

    @property
    def strname(self):
        """Convenience method for getting the string name"""
        return self.name.safe_substitute()

    def __repr__(self):
        return "Dashboard({} [id:{}])".format(self.strname, self.dash_id)

    def to_dict(self):
        """Convert to dictionary for creating"""
        return {"name": self.strname}

class Graph(object):
    # pylint: disable=attribute-defined-outside-init
    """Instrumental graph"""
    @staticmethod
    def new(title, metrics, units="",
            stacked=False, continuous=False, dash_id=None, graph_id=None, **values):
        # pylint: disable=too-many-arguments
        """Similar to an __init__, but can use copy to return multiple objects"""

        base = Graph()

        base.title = string.Template(title)
        base.metrics = metrics
        base.units = units
        base.stacked = stacked
        base.continuous = continuous
        base.dash_id = dash_id
        base.graph_id = graph_id

        # if no values are given, return the base (this also prevents infinite copy recursion)
        if not values:
            return [base]

        # otherwise, return copy of the base
        return base.copy(**values)

    def delete(self):
        """Delete the graph"""
        resp = _api("DELETE", "/graphs/{}".format(self.graph_id))
        return resp

    def create(self):
        """Create a new graph"""
        resp = _api("POST", "/graphs/", data=self.to_dict()).json()
        try:
            self.graph_id = resp["id"]
        except TypeError as e:
            print(resp)
            raise e

    def sub(self, **values):
        """Implement `sub` for `cross_product_copy`"""

        # call sub on metrics, making sure not to modify the original
        metrics = [metric.sub(**values) for metric in self.metrics]
        # NOTE: new with no values is guaranteed to return one object
        return Graph.new(
            self.title.safe_substitute(values), metrics, self.units, self.stacked,
            self.continuous, self.dash_id, self.graph_id).pop()

    def copy(self, **values):
        """Copy using `cross_product_copy`, also copying all metrics and passing down values to the
        metrics"""

        # # call sub to make a new graph so we don't modify the original
        new_graph = self.sub()

        # # copy graph
        values = clean_keys(values, self.title)
        return cross_product_copy(new_graph, **values)

    def update_diff(self, other, update_self=True):
        """Updates the graph and returns a diff dict of field values in the other graph that are
        different from self

        If update_self is set to False, only return the diff

        Thre returned diff is used to determine if an update is required, and to produce
        confirmation messages before applying changes
        """

        diffs = {}
        dict_self = self.to_dict()
        dict_other = other.to_dict()

        for k, v in dict_self.items():
            # skip title, dashboard_id
            if k in {"title", "dashboard_id"}:
                continue

            if v != dict_other[k]:
                diffs[k] = (v, dict_other[k])

        # update fields
        if update_self and diffs:
            for k in diffs:
                if k == "metrics":
                    self.metrics = other.metrics
                elif k == "units":
                    self.units = other.units
                elif k == "stacked":
                    self.stacked = other.stacked
                elif k == "continuous":
                    self.continuous = other.continuous
                else:
                    continue
            diffs["title"] = self.strtitle
            diffs["dash_id"] = self.dash_id

        return diffs

    def update(self):
        """Updates the graph"""
        resp = _api("PATCH", "/graphs/{}".format(self.graph_id), data=self.to_dict())
        return resp

    @staticmethod
    def get_all():
        """Get all graphs"""
        graphs = []

        resp = _api("GET", "/graphs").json()
        for graph in resp:
            graphs.append(Graph.new(
                title=graph["title"],
                metrics=[Metric.new(metric).pop() for metric in graph["metrics"]],
                units=graph["units"],
                stacked=graph["stacked"],
                continuous=graph["continuous"],
                dash_id=graph["dashboard_id"],
                graph_id=graph["id"]
                ).pop())

        return graphs

    @property
    def strtitle(self):
        """Convenience method for getting the string title"""
        return self.title.safe_substitute()

    def __repr__(self):
        return "Graph({} [dash:{}])".format(self.strtitle, self.dash_id)

    def to_dict(self):
        """Convert to dictionary for creating"""

        return {
            "title": self.strtitle,
            "dashboard_id": self.dash_id,
            "metrics": [m.strmetric for m in self.metrics],
            "units": self.units,
            "stacked": self.stacked,
            "continuous": self.continuous,
            }

class Metric(object):
    # pylint: disable=attribute-defined-outside-init
    """Instrumental graph metric"""

    @staticmethod
    def new(metric, **values):
        """Similar to an __init__, but can use copy to return multiple objects"""
        base = Metric()
        base.metric = string.Template(metric)

        # if no values are given, return the base (this also prevents infinite copy recursion)
        if not values:
            return [base]

        # otherwise, return copy of the base
        return base.copy(**values)

    def sub(self, **values):
        """Implement `sub` for `cross_product_copy`"""

        # NOTE: new with no values is guaranteed to return one object
        return Metric.new(self.metric.safe_substitute(values)).pop()

    def copy(self, **values):
        """Copy using `cross_product_copy`"""

        values = clean_keys(values, self.metric)
        return cross_product_copy(self, **values)

    @property
    def strmetric(self):
        """Convenience method for getting the string metric"""
        return self.metric.safe_substitute()

    def __str__(self):
        return self.strmetric

    def __repr__(self):
        return "Metric({})".format(self.strmetric)

def cross_product_copy(template_copiable, **values):
    """Copy the object by substituting values, doing a cross-product when the value is a list

    By cross-product, we mean that if one of the values is a list, this method is called
    recursively with the key being mapped to each item in the list, resulting in returning as many
    copies of the object as the length of the list. An example to illustrate:

    >>> cross_product_copy(obj('$field1 $field2'), field1=['a', 'b'], field2=1)
    [obj('a 1'), obj('b 1')]

    The first argument, template_copiable, refers to an object which has a method `sub(self,
    **values)` that uses string.Template.safe_substitute to copy its templated values and make
    copies of its children, also using the provided values. The `values` dict passed to `sub`
    should never expect to have any values that are lists; `cross_product_copy` ensures this.
    """

    # start with one copy
    copies = [template_copiable]

    for k, v in values.items():
        # new list of created copies
        new_copies = []
        # apply to all copies
        for c in copies:
            # do cross product if value is a list
            if isinstance(v, list):
                for i in v:
                    new_copies.extend(cross_product_copy(c, **{k: i}))
            else:
                # call object's sub
                new_copies.append(c.sub(**{k: v}))

        # replace list for the next k-v pair
        copies = new_copies

    return copies

def clean_keys(keys, *templates):
    """Remove keys that don't appear in the templates"""
    cleaned = {}
    for k, v in keys.items():
        for template in templates:
            if "$"+k in template.template:
                cleaned[k] = v

    return cleaned

def update(new_dashboards):
    """Update the existing dashboards and graphs with new ones read from the yaml file"""

    # get current dashboards and graphs
    old_dashboards = Dashboard.get_all()
    old_graphs = Graph.get_all()

    # create map of old dashboard names to dashboards and dashboard ids to names
    dashmap = {}
    dash_idmap = {}
    for dash in old_dashboards:
        dashmap[dash.strname] = dash
        dash_idmap[dash.dash_id] = dash.strname

    # create nested map of old dashboard names to old graph titles to graphs
    graphmap = {}
    for graph in old_graphs:
        dashname = dash_idmap[graph.dash_id]
        if dashname not in graphmap:
            graphmap[dashname] = {}
        graphmap[dashname][graph.strtitle] = graph

    # track create/delete/updates
    dashboards_to_create = []
    dashboards_to_delete_names = set(dashmap.keys())
    graphs_to_create = []
    graphs_to_update = []
    graph_diffs = []
    graphs_to_delete_titles = {
        dashname: set(graphs.keys()) for dashname, graphs in graphmap.items()}

    # check for dashboard updates
    for dash in new_dashboards:
        # create if dashboard does not exist
        if dash.strname not in dashmap:
            dashboards_to_create.append(dash)
            # also create graphs
            for graph in dash.graphs:
                graphs_to_create.append(graph)
        # if it exists, don't delete
        else:
            dashboards_to_delete_names.remove(dash.strname)

            # check graph updates
            for graph in dash.graphs:
                # create if graph does not exist
                if graph.strtitle not in graphmap[dash.strname]:
                    # set dash id for graph
                    graph.dash_id = dashmap[dash.strname].dash_id
                    graphs_to_create.append(graph)
                # if it exists, don't delete
                else:
                    graphs_to_delete_titles[dash.strname].remove(graph.strtitle)
                    # check if it needs to be updated
                    old_graph = graphmap[dash.strname][graph.strtitle]
                    diff = old_graph.update_diff(graph)
                    if diff:
                        graphs_to_update.append(old_graph)
                        graph_diffs.append(diff)

    for dash in dashboards_to_create:
        dash.dash_id = 123
        for graph in dash.graphs:
            graph.dash_id = dash.dash_id

    # list of dashboards and graphs to delete
    dashboards_to_delete = [dashmap[name] for name in dashboards_to_delete_names]
    graphs_to_delete = []
    for dashname, titles in graphs_to_delete_titles.items():
        for title in titles:
            graph = graphmap[dashname][title]
            graphs_to_delete.append(graph)

    print_preview(
        dashboards_to_create,
        graphs_to_create,
        graph_diffs,
        dashboards_to_delete,
        graphs_to_delete,
        dash_idmap)

    # Prompt confirmation
    confirm = input("\nAre you sure you want to make these changes? (yes): ") == "yes"
    if confirm:
        # create dashboards (this also updates the dash_id of its children graphs)
        print("\nCreating dashboards")
        for dash in dashboards_to_create:
            dash.create()
            print('.', end='', flush=True)
        print("\nCreating graphs")
        # create graphs
        for graph in graphs_to_create:
            graph.create()
            print('.', end='', flush=True)
        # update graphs
        print("\nUpdating graphs")
        for graph in graphs_to_update:
            graph.update()
            print('.', end='', flush=True)
        # delete graphs (do this first since deleting the dashboard will delete its graphs)
        print("\nDeleting graphs")
        for graph in graphs_to_delete:
            graph.delete()
            print('.', end='', flush=True)
        # delete dashboards
        print("\nDeleting dashboards")
        for dash in dashboards_to_delete:
            dash.delete()
            print('.', end='', flush=True)
        print("\nCompleted")
    else:
        print("Aborted")

def print_preview(new_dashes, new_graphs, updated_graphs, deleted_dashes, deleted_graphs,
        dash_idmap):
    """Print a preview of changes to be made"""

    print("The following changes will be made:")

    print("\nNew dashboards\n")
    for dash in new_dashes:
        print(dash)

    print("\nNew graphs\n")
    for graph in new_graphs:
        dashname = dash_idmap[graph.dash_id]
        print("[{}] {}".format(dashname, graph))

    print("\nUpdated graphs\n")
    for df in updated_graphs:
        title = df.pop("title")
        dashname = dash_idmap[df.pop("dash_id")]
        for k, v in df.items():
            print("[{}][{}]".format(dashname, title), k+":", v[0], "->", v[1])

    print("\nDashboards to be deleted\n")
    for dash in deleted_dashes:
        print(dash)

    print("\nGraphs to be deleted\n")
    for graph in deleted_graphs:
        dashname = dash_idmap[graph.dash_id]
        print("[{}] {}".format(dashname, graph))

if __name__ == "__main__":
    # read from file (get filename if it exists)
    if len(sys.argv) == 2:
        data = read_yaml.read(sys.argv[1])
    else:
        data = read_yaml.read()

    update(data)
