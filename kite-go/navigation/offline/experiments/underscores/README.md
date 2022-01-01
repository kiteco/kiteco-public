# Should shingles cross over underscores?

2020-12-10

## Background

We compare two approaches of transforming words with underscores into shingles.

For the control, we convert a word into shingles by finding any five consecutive letters.
So `foobar` becomes two shingles: `fooba` and `oobar`.
And `foo_barbaz` also becomes two shingles: `barba` and `arbaz`.

For the treatment, we also allow shingles which span across underscores.
So `foobar` still becomes two shingles: `fooba` and `oobar`.
But `foo_barbaz` becomes five shingles: `fooba`, `oobar`, `obarb`, `barba`, and `arbaz`.

## Instructions

To reproduce the experiment: `make all`

## Validation

### Control

label|file_f1|file_precision|file_recall|line_f1|line_precision|line_recall
-|:-:|:-:|:-:|:-:|:-:|:-:
angular/angular|0.2408|0.2506|0.3609|0.2154|0.2276|0.3675
apache/airflow|0.2562|0.247|0.4547|0.1653|0.1889|0.2504
apache/hive|0.2433|0.2523|0.4137|0.1506|0.1582|0.2915
apache/spark|0.2132|0.2152|0.3151|0.1562|0.158|0.2551
django/django|0.2765|0.2451|0.4609|0.1657|0.1666|0.2549
facebook/react|0.2404|0.2734|0.363|0.146|0.1368|0.3287
prestodb/presto|0.2344|0.3235|0.2464|0.1899|0.2027|0.2982
rails/rails|0.2614|0.2131|0.5009|0.1674|0.1656|0.2588
spring-projects/spring-framework|0.2277|0.1725|0.4749|0.1598|0.1527|0.2471
tensorflow/tensorflow|0.2014|0.1874|0.3808|0.1313|0.15|0.1971
mean|0.2395|0.238|0.3971|0.1648|0.1707|0.2749

### Treatment

label|file_f1|file_precision|file_recall|line_f1|line_precision|line_recall
-|:-:|:-:|:-:|:-:|:-:|:-:
angular/angular|0.2412|0.2515|0.3612|0.2142|0.2266|0.3653
apache/airflow|0.255|0.244|0.4616|0.1702|0.195|0.2567
apache/hive|0.2457|0.2561|0.4155|0.1501|0.1577|0.2947
apache/spark|0.2152|0.2175|0.3167|0.1581|0.1594|0.2572
django/django|0.2791|0.2466|0.4652|0.1641|0.1682|0.2463
facebook/react|0.2395|0.2743|0.3581|0.1449|0.1365|0.3267
prestodb/presto|0.2333|0.3225|0.2445|0.1899|0.2035|0.2995
rails/rails|0.2622|0.2148|0.4973|0.1713|0.1765|0.257
spring-projects/spring-framework|0.2298|0.1741|0.4806|0.1611|0.1511|0.2496
tensorflow/tensorflow|0.2081|0.1936|0.3904|0.1342|0.1637|0.2007
mean|0.2409|0.2395|0.3991|0.1658|0.1738|0.2754

## Examples

Note: these are cherry-picked examples, not random samples.

For the treatment and control, we get recommendations based on the current path.
We find the ranking (1-based) of the related path and call that the "related path rank".
We find the highest scoring keywords from the related path.


### Example 1

#### Input
- Current path: [`sidebar/src/components/WindowMode/index.tsx`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/components/WindowMode/index.tsx)
- Related path: [`sidebar/src/components/WindowMode/index.module.css`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/components/WindowMode/index.module.css)

#### Related path rank
- Control: `8`
- Treatment: `1`

#### Related path keywords
|Control|Treatment|
|-|-|
|`sidebar__tooltip__paragraph`|`sidebar__icon__window`|
|`sidebar__tooltip__title`|`sidebar__tooltip__paragraph`|
|`sidebar__tooltip`|`sidebar__tooltip__title`|
|`sidebar__icon__window`|`sidebar__tooltip`|
|`window`|`help__option`|


### Example 2

#### Input
- Current path: [`sidebar/src/containers/Logs.js`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/containers/Logs.js)
- Related path: [`sidebar/src/assets/logs.css`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/assets/logs.css)

#### Related path rank
- Control: `112`
- Treatment: `2`

#### Related path keywords
|Control|Treatment|
|-|-|
|`disabled`|`logs__link`|
|`row__description`|`logs__cta`|
|`logs__section`|`logs__section`|
|`pointer`|`row__description`|
|`decoration`|`row__cta`|


### Example 3

#### Input
- Current path: [`sidebar/src/containers/Examples/assets/code-example.css`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/containers/Examples/assets/code-example.css)
- Related path: [`sidebar/src/containers/Examples/components/CodeExample.js`](https://github.com/kiteco/kiteco/blob/master/sidebar/src/containers/Examples/components/CodeExample.js)

#### Related path rank
- Control: `89`
- Treatment: `1`

#### Related path keywords
|Control|Treatment|
|-|-|
|`example__postlude`|`example__title__wrapper`|
|`highlightedIdentifier`|`examples__code`|
|`example__prelude`|`example__postlude`|
|`postlude`|`example__prelude`|
|`example__title__wrapper`|`example__title`|


### Example 4

#### Input
- Current path: [`kite-python/analysis/conversion-model/model.py`](https://github.com/kiteco/kiteco/blob/master/kite-python/analysis/conversion-model/model.py)
- Related path: [`kite-go/client/internal/conversion/monetizable/model/model.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/conversion/monetizable/model/model.go)

#### Related path rank
- Control: `11`
- Treatment: `2`

#### Related path keywords
|Control|Treatment|
|-|-|
|`FillUnknownIntelliJInstalled`|`FillUnknownIntelliJPaid`|
|`IntelliJInstalled`|`IntelliJPaid`|
|`intellij_installed`|`intellij_paid`|
|`logistic`|`FillUnknownIntelliJInstalled`|
|`FillUnknownPyCharmInstalled`|`FillUnknownCPUThreads`|


### Example 5

#### Input
- Current path: [`kite-go/lang/python/pythongraph/graph.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph/graph.go)
- Related path: [`kite-python/kite_ml/kite/graph_data/graph.py`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_ml/kite/graph_data/graph.py)

#### Related path rank
- Control: `7`
- Treatment: `1`

#### Related path keywords
|Control|Treatment|
|-|-|
|`EdgeType`|`AST_CHILD_ASSIGN_RHS`|
|`NodeType`|`ast_child_assign_rhs`|
|`VariableID`|`AST_CHILD_ARG_VALUE`|
|`_forward`|`ast_child_arg_value`|
|`forward`|`AST_CHILD_ATTR_VALUE`|


### Example 6

#### Input
- Current path: [`kite-go/lang/python/pythongraph/graph.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph/graph.go)
- Related path: [`kite-go/lang/python/pythongraph/rendered/templates/graph.html`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph/rendered/templates/graph.html)

#### Related path rank
- Control: `337`
- Treatment: `4`

#### Related path keywords
|Control|Treatment|
|-|-|
|`variable_usage_node`|`variable_usage_node`|
|`styledEdges`|`last_lexical_use`|
|`computed_from`|`ast_internal_node`|
|`edges`|`ast_teminal_node`|
|`last_lexical_use`|`computed_from`|


### Example 7

#### Input
- Current path: [`kite-python/kite_emr/kite/emr/bundle.py`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_emr/kite/emr/bundle.py)
- Related path: [`kite-python/kite_emr/kite/emr/constants.py`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_emr/kite/emr/constants.py)

#### Related path rank
- Control: `45`
- Treatment: `1`

#### Related path keywords
|Control|Treatment|
|-|-|
|`BUNDLE_DIR`|`KITE_PYTHON`|
|`bundle`|`BUNDLE_DIR`|
|`KITE_EMR_BUCKET`|`KITE_EMR_BUCKET`|
|`environ`|`KITE_EMR_ROOT`|
|`KITE_PYTHON`|`kite_emr`|

## Conclusion

We should switch to the treatment, because:
- The overall validation metrics are marginally better.
- We know of several interesting examples where the treatment performs significantly better.
