# Instrumental YAML Spec

This spec documents the format of the YAML file that defines the dashboards and graphs on instrumental.

## Overview

The document has three sections, detailed below: dashboards, dashboard templates, and graph templates. The templates can be used to conveniently replicate dashboards and graphs that use similar structures, but with slightly different parameters. The dashboards section defines the actual dashboards and graphs that will be put on Instrumental, and can use templates.

## Templates and cross-product replication

Dashboards, graphs, and metrics each have a field that can accept a substitution token, denoted using the `$` symbol - for example, a dashboard name with a substitutable `region` token could be called `"$region Dashboard"`. The substitution token name should be snake-cased.

The values of these substitution tokens can be specified as part of the item definition. Any values in the item definition that are not a field of the item itself becomes values for a substitution token.

```
dashboards:
  - "$region Dashboard":
      region: "us-west-1"
```

In the above example, because dashboards have no fields other than its title, any values in its definition map will be treated as substitution values.

Substitution values can be inherited by an item's children. For example, if a `region` field is defined in a dashboard, and both its graphs and the graphs' metrics have the `$region` token, all children will inherit the value defined by the dashboard.

Substitution values can be either a single string or a list of strings. If it is a single string, it will simply replace the corresponding token in the item. However, if it is defined as a list of strings, it will cause the item to _cross-product replicate_.

In cross-product replication, a new item will be created for each value in each substitution value field that is defined as a list. If there are more than one list, it will create a new item for every combination of substitution values.

```
dashboards:
  - "$region $node $metric Dashboard":
      region:
        - "us-west-1"
        - "us-east-1"
      node:
        - "user-node"
        - "user-mux"
      metric: "systems"
```

The above definition will create **4** dashboards, one for each combination of the `$region` and `$node` tokens, while the `$metric` token is individually filled in on each dashboard, since it is a single value.

Regarding how cross-product replication interacts with substitution inheritence, the cross-product is only applied to the item that the list values are defined in; its children do not inherit the entire list, instead inheriting the single value given to each of the copies of the parent item.

```
dashboards:
  - "$region Dashboard":
      region:
        - "us-west-1"
        - "us-east-1"
      graphs:
        - "$region Graph"
```

The above example will produce _two_ dashboards, each with _one_ graph: a "us-west-1 Dashboard" with "us-west-1 Graph" and a "us-east-1 Dashboard" with "us-east-1 Graph".

## Notes on YAML

- generally speaking, put quotes around all string values that are multiple words and/or have separators like `-` and `.`
  - consider also quoting single words that are in the same list as other quoted strings
- when defining a map in a list, note that there are _two_ indentations between the `-` for the list and the fields of the map:
    ```
    maps:
      - item1:
          k1: v1
          k2: v2
      - item2:
          ...
    ```

## Top level items

The top level items allowed are `dashboards`, `dashboard_templates`, and `graph_templates`. `dashboards` is required, while the others are optional.

### Dashboards

```
dashboards:
  - [name]:
      k1: v1
      k2: v2
      ...
      graphs:
        - [graph1]
        ...
  - [template key]:
      k1: v1
      ...
```

- items under `dashboards` must be a list of maps
- `name` is the name of the dashboard, specified as a key of the map
- `name` can take substitute values
- `graphs` is a list of either maps or strings
  - if it is a string, it should be the template key for a graph template
  - refer to [Graphs](#graphs) for the map format
- `graphs` is not required, but generally it is unhelpful to have a dashboard without graphs
- if using a template, specify the key as `t_` + the template key, e.g. `t_dashboard1` for the `dashboard1` template

### Graphs
```
...
graphs:
  - [title]:
      units: [unit]
      stacked: [true/false]
      continuous: [true/false]
      k1: v1
      k2: v2
      ...
      metrics:
        - [metric]
        ...
  - [template key]:
      k1: v1
      ...
```

- graphs can be specified as either
  - a new graph with the title as the key to a map of fields
  - a template-based graph with the template key as the key to a map of fields
  - a template-based graph without fields, specified as just the template key
- `title` is the title of the graph, specified as a key of the map
- `title` can take substitute values
- `metrics` is a list of either maps or strings, depending on whether the metric specifies its own substitution values or inherits them from the graph/dashboard
- `metrics` is not required, but generally it is unhelpful to have a graph without metrics
- if using a template, specify the key as `t_` + the template key, e.g. `t_graph1` for the `graph1` template

### Metrics
```
...
metrics:
  - [metric1]
  - [metric2]:
      k1: v1
      k2: v2
      ...
```
- metrics can be specified as either a string value for the metric string or as the key to a map of substitution values
- the metric string can take substitute values

### Templates
```
graph_templates:
  [template key]:
    title: [graph title]
    ...
  ...

dashboard_templates:
  [template key]:
    name: [dashboard name]
    ...
  ...
```

- items under templates must be a map
- instead of specifying the name/title as the key to the map, the map key is instead the template key used by items under `dashboard` to refer to the template, and the name/title is specified as a field in the map
