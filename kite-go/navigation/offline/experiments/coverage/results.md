# Histograms

Note validation data leaks into training data when using commits.

![](histogram.png)

# Files using commits

## [`kite-go/navigation/recommend/recommend.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend.go)

Coverage: 1.250059

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/navigation/recommend/recommend_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend_test.go)|0.05263157894736842|
|1|2|0.250000|[`kite-go/navigation/recommend/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/load_test.go)|0.05263157894736842|
|2|15|0.000031|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|0.05263157894736842|
|3|16|0.000015|[`kite-go/navigation/offline/validation/validation_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/validation_test.go)|0.05263157894736842|
|4|17|0.000008|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|0.05263157894736842|
|5|18|0.000004|[`kite-go/navigation/recommend/vectorizer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/vectorizer_test.go)|0.05263157894736842|
|6|19|0.000002|[`kite-go/navigation/recommend/graph_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/graph_test.go)|0.05263157894736842|
|7|24|0.000000|[`kite-go/navigation/codebase/project_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/project_test.go)|0.05263157894736842|
|8|27|0.000000|[`kite-go/navigation/git/git_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/git_test.go)|0|
|9|29|0.000000|[`kite-go/navigation/metrics/metrics_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/metrics/metrics_test.go)|0|
|10|32|0.000000|[`kite-go/navigation/ignore/ignore_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/ignore/ignore_test.go)|0|
|11|42|0.000000|[`kite-go/navigation/localpath/localpath_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/localpath/localpath_test.go)|0|
|12|48|0.000000|[`kite-go/navigation/recommend/files_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/files_test.go)|0.05263157894736842|
|13|51|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0.05263157894736842|
|14|59|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0|
|15|64|0.000000|[`kite-go/client/internal/settings/compat_manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/compat_manager_test.go)|0|
|16|112|0.000000|[`kite-golib/lexicalv0/text/split_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/split_test.go)|0|
|17|129|0.000000|[`kite-go/client/internal/clientapp/test/license_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test/license_test.go)|0|
|18|131|0.000000|[`kite-go/client/internal/kitelocal/internal/indexing/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/indexing/manager_test.go)|0|
|19|134|0.000000|[`kite-go/clientlogs/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs/server_test.go)|0|
|20|136|0.000000|[`kite-go/client/internal/kitelocal/internal/filesystem/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/filesystem/manager_test.go)|0|
|21|175|0.000000|[`kite-go/navigation/git/storage_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/storage_test.go)|0|
|22|186|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0|
|23|192|0.000000|[`kite-go/community/account/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/community/account/manager_test.go)|0|
|24|200|0.000000|[`local-pipelines/mixing/data/normalize/analyze_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/mixing/data/normalize/analyze_test.go)|0|

## [`kite-go/client/internal/kitelocal/internal/navigation/manager.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager.go)

Coverage: 1.132828

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|15.134642160957952|
|1|3|0.125000|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|3.3065998329156225|
|2|7|0.007812|[`kite-go/navigation/codebase/project_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/project_test.go)|1.556599832915622|
|3|16|0.000015|[`kite-go/navigation/recommend/recommend_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend_test.go)|0.05263157894736842|
|4|23|0.000000|[`kite-go/client/internal/status/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/status/manager_test.go)|0.18518518518518517|
|5|25|0.000000|[`kite-go/client/internal/settings/compat_manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/compat_manager_test.go)|0.35185185185185186|
|6|53|0.000000|[`kite-go/client/internal/ws/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/ws/manager_test.go)|0.1111111111111111|
|7|59|0.000000|[`kite-go/client/internal/metrics/livemetrics/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/manager_test.go)|0.018518518518518517|
|8|75|0.000000|[`kite-go/navigation/git/git_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/git_test.go)|0.08333333333333333|
|9|76|0.000000|[`kite-golib/wstest/helpers.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/wstest/helpers.go)|0.1111111111111111|
|10|78|0.000000|[`kite-go/client/internal/autocorrect/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autocorrect/manager_test.go)|0|
|11|84|0.000000|[`kite-go/client/internal/clientapp/test/mainloop_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test/mainloop_test.go)|0|
|12|91|0.000000|[`kite-go/navigation/recommend/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/load_test.go)|0.05263157894736842|
|13|97|0.000000|[`kite-go/client/internal/desktoplogin/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/desktoplogin/manager_test.go)|0|
|14|100|0.000000|[`kite-go/client/internal/kitelocal/test/indexing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/indexing_test.go)|0|
|15|101|0.000000|[`kite-go/clientlogs/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs/server_test.go)|0|
|16|103|0.000000|[`kite-go/client/internal/kitelocal/test/autosearch.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/autosearch.go)|0|
|17|112|0.000000|[`kite-go/navigation/offline/validation/validation_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/validation_test.go)|0.05263157894736842|
|18|117|0.000000|[`kite-go/client/internal/kitelocal/event_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/event_processor_test.go)|0|
|19|121|0.000000|[`kite-go/client/internal/clientapp/test_setup.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test_setup.go)|0|
|20|123|0.000000|[`kite-go/client/internal/settings/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/manager_test.go)|0|
|21|138|0.000000|[`kite-go/client/internal/autosearch/test/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autosearch/test/manager_test.go)|0|
|22|140|0.000000|[`kite-go/client/internal/conversion/cohort/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/conversion/cohort/manager_test.go)|0|
|23|149|0.000000|[`kite-go/client/internal/kitelocal/permissions/handlers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/handlers_test.go)|0|
|24|152|0.000000|[`kite-go/client/internal/kitelocal/test/editorapi_buffer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/editorapi_buffer_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/navigation/offline/validation/load_test.go|0.05263157894736842|
|kite-go/navigation/recommend/files_test.go|0.05263157894736842|
|kite-go/navigation/recommend/graph_test.go|0.05263157894736842|
|kite-go/navigation/recommend/vectorizer_test.go|0.05263157894736842|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go|0.14285714285714285|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go|0.14285714285714285|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go|0.14285714285714285|
|kite-go/navigation/ignore/ignore_test.go|0.25|
|kite-go/navigation/ignore/munge_test.go|0.25|
|kite-go/client/internal/conversion/cohort/autostart-trial_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/cta_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/quiet-autostart_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/usage-paywall_test.go|0.018518518518518517|
|kite-go/client/internal/kitelocal/internal/signatures/manager_test.go|0.018518518518518517|
|kite-go/client/internal/metrics/proselected_test.go|0.018518518518518517|
|kite-go/client/internal/notifications/manager_test.go|0.018518518518518517|
|local-pipelines/lexical/train/cmds/utils/datasets_test.go|0.018518518518518517|
|kite-go/navigation/git/cache_test.go|0.08333333333333333|
|kite-go/navigation/git/storage_test.go|0.08333333333333333|

## [`kite-go/lang/language.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/language.go)

Coverage: 1.195325

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/lang/language_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/language_test.go)|1.0|
|1|3|0.125000|[`local-pipelines/lexical/train/cmds/utils/datasets_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/lexical/train/cmds/utils/datasets_test.go)|0.8333333333333333|
|2|4|0.062500|[`kite-golib/lexicalv0/langgroup_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/langgroup_test.go)|0|
|3|7|0.007812|[`kite-go/client/internal/kitelocal/permissions/support_status_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/support_status_test.go)|1.0|
|4|17|0.000008|[`kite-go/client/internal/metrics/livemetrics/languagemetrics_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/languagemetrics_test.go)|0|
|5|18|0.000004|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|6|20|0.000001|[`kite-golib/lexicalv0/encoder_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/encoder_test.go)|0.5|
|7|35|0.000000|[`kite-go/navigation/localpath/localpath_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/localpath/localpath_test.go)|0|
|8|45|0.000000|[`kite-go/client/internal/clientapp/lang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/lang_test.go)|0|
|9|57|0.000000|[`kite-golib/lexicalv0/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/context_test.go)|0|
|10|58|0.000000|[`kite-golib/lexicalv0/inspect/inspect_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/inspect/inspect_test.go)|0|
|11|63|0.000000|[`kite-golib/lexicalv0/text/split_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/split_test.go)|0.29166666666666663|
|12|67|0.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|0|
|13|76|0.000000|[`kite-golib/lexicalv0/text/lexer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/lexer_test.go)|0.125|
|14|78|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go)|0|
|15|80|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0|
|16|86|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go)|0|
|18|90|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|19|95|0.000000|[`kite-golib/lexicalv0/javascript/render_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript/render_test.go)|0|
|20|96|0.000000|[`kite-go/client/internal/kitelocal/permissions/handlers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/handlers_test.go)|0|
|21|97|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0|
|22|114|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go)|0|
|23|120|0.000000|[`kite-go/curation/titleparser/utility_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/curation/titleparser/utility_test.go)|0|
|24|131|0.000000|[`kite-go/lang/detect_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/detect_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-golib/lexicalv0/text/render_test.go|0.125|

## [`kite-go/client/internal/kitelocal/internal/completions/lexical.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/completions/lexical.go)

Coverage: 0.000368

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|12|0.000244|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|1|13|0.000122|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0|
|2|19|0.000002|[`kite-golib/complete/data/api_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/api_test.go)|0|
|3|29|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0|
|4|34|0.000000|[`kite-go/client/internal/settings/compat_manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/compat_manager_test.go)|0.11851851851851852|
|5|40|0.000000|[`kite-go/client/internal/metrics/completions/metric_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/completions/metric_test.go)|0|
|6|41|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0|
|7|42|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|0|
|8|48|0.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|0.018518518518518517|
|9|49|0.000000|[`kite-go/client/internal/kitelocal/event_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/event_processor_test.go)|0.14285714285714285|
|10|51|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0|
|11|53|0.000000|[`kite-go/client/internal/metrics/livemetrics_test/metrics_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics_test/metrics_test.go)|0.14285714285714285|
|12|55|0.000000|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0|
|13|64|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0|
|14|66|0.000000|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0|
|15|68|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|16|77|0.000000|[`kite-go/clientlogs/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs/server_test.go)|0|
|17|79|0.000000|[`kite-go/client/internal/notifications/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/notifications/manager_test.go)|0.018518518518518517|
|18|83|0.000000|[`kite-go/client/platform/platform_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/platform/platform_test.go)|0|
|19|84|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0|
|20|87|0.000000|[`kite-go/client/internal/metrics/livemetrics/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/manager_test.go)|0.018518518518518517|
|21|93|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|22|96|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0|
|23|98|0.000000|[`kite-golib/lexicalv0/cmds/test_inference_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_inference_latency/main.go)|0|
|24|102|0.000000|[`kite-go/client/internal/kitelocal/test/editorapi_buffer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/editorapi_buffer_test.go)|0.14285714285714285|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/client/internal/conversion/cohort/autostart-trial_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/cta_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/quiet-autostart_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/usage-paywall_test.go|0.018518518518518517|
|kite-go/client/internal/kitelocal/internal/signatures/manager_test.go|0.018518518518518517|
|kite-go/client/internal/metrics/proselected_test.go|0.018518518518518517|
|kite-go/client/internal/status/manager_test.go|0.018518518518518517|
|local-pipelines/lexical/train/cmds/utils/datasets_test.go|0.018518518518518517|

## [`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/Data_inputs.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/Data_inputs.go)

Coverage: 0.265632

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|2|0.250000|[`kite-golib/lexicalv0/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/context_test.go)|1.342156862745098|
|1|6|0.015625|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|1.2509803921568627|
|2|18|0.000004|[`kite-golib/lexicalv0/langgroup_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/langgroup_test.go)|0|
|3|19|0.000002|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0.1801470588235294|
|4|20|0.000001|[`local-pipelines/lexical/train/cmds/utils/datasets_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/lexical/train/cmds/utils/datasets_test.go)|0|
|5|24|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0.8333333333333333|
|6|25|0.000000|[`kite-golib/lexicalv0/cmds/test_search_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_search_latency/main.go)|0.24136321195144722|
|7|29|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0.6333333333333333|
|8|30|0.000000|[`kite-go/client/internal/kitelocal/permissions/support_status_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/support_status_test.go)|0|
|9|31|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0.3333333333333333|
|10|36|0.000000|[`kite-go/client/internal/clientapp/lang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/lang_test.go)|0|
|11|37|0.000000|[`kite-golib/lexicalv0/cmds/test_inference_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_inference_latency/main.go)|0.13025210084033612|
|12|38|0.000000|[`kite-golib/lexicalv0/encoder_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/encoder_test.go)|0.33723262032085566|
|13|43|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0.058823529411764705|
|14|44|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textc_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textc_test.go)|0.43333333333333335|
|15|45|0.000000|[`kite-golib/lexicalv0/predict/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/predict/context_test.go)|0|
|16|54|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0|
|17|56|0.000000|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0|
|18|61|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0|
|19|71|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/curated_context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/curated_context_test.go)|0.058823529411764705|
|20|72|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|21|73|0.000000|[`kite-golib/lexicalv0/text/lexer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/lexer_test.go)|0.7588235294117647|
|22|77|0.000000|[`kite-go/lang/lexical/lexicalcomplete/api/events_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/api/events_test.go)|0|
|23|82|0.000000|[`kite-go/lang/python/pythoncomplete/performancetest/testenv.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/performancetest/testenv.go)|0|
|24|90|0.000000|[`kite-golib/lexicalv0/cmds/test_prettify_golang/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_prettify_golang/main.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-golib/lexicalv0/javascript/javascript_test.go|0.24632352941176472|
|kite-golib/lexicalv0/javascript/treesitter_test.go|0.24632352941176472|
|kite-golib/lexicalv0/text/split_test.go|0.24632352941176472|
|kite-golib/lexicalv0/inspect/inspect_test.go|0.09090909090909091|
|kite-go/client/internal/metrics/completions/metric_test.go|0.058823529411764705|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/agnostic_test.go|0.058823529411764705|
|kite-golib/lexicalv0/javascript/render_test.go|0.058823529411764705|
|kite-golib/lexicalv0/cmds/compare_text_model/normalize_test.go|0.058823529411764705|
|kite-go/client/internal/status/manager_test.go|0.058823529411764705|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/autoCloseParen.go|0.5333333333333333|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/noNonMatchedCloseParen.go|0.5333333333333333|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/dedupeCall.go|0.2|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/identifierMiddleOfLine.go|0.2|
|kite-golib/lexicalv0/text/render_test.go|0.2|
|kite-golib/lexicalv0/css/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/html/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/python/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/vue/lexer_test.go|0.1213235294117647|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/basic1.js|0.1|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/texthtml_test.go|0.1|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjava_test.go|0.1|
|kite-golib/lexicalv0/predict/partialrun_test.go|0.1|

## [`kite-go/lang/python/pythoncomplete/driver/mixing.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing.go)

Coverage: 0.002443

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|9|0.001953|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0.4793939393939394|
|1|11|0.000488|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0.022727272727272728|
|2|19|0.000002|[`kite-golib/complete/corpustests/testcase.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/corpustests/testcase.go)|0.6|
|3|23|0.000000|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0.14606060606060606|
|4|26|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|5|27|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0.022727272727272728|
|6|28|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/lexical_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/lexical_test.go)|0.022727272727272728|
|7|41|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0.12333333333333332|
|8|43|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0.022727272727272728|
|9|44|0.000000|[`kite-go/lang/python/pythoncomplete/performancetest/testenv.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/performancetest/testenv.go)|0.14606060606060606|
|10|54|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/empty_calls_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/empty_calls_test.go)|0.022727272727272728|
|11|55|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|0|
|12|59|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/corpus/lexical.py`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/corpus/lexical.py)|0.8333333333333333|
|13|60|0.000000|[`kite-golib/complete/data/completion_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/completion_test.go)|0|
|14|71|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/attributes_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/attributes_test.go)|0.022727272727272728|
|15|83|0.000000|[`kite-go/client/internal/metrics/completions/metric_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/completions/metric_test.go)|0|
|16|86|0.000000|[`kite-go/client/internal/settings/compat_manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/compat_manager_test.go)|0.2|
|17|87|0.000000|[`kite-golib/complete/data/api_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/api_test.go)|0|
|18|90|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/ggnn_provider_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/ggnn_provider_test.go)|0.022727272727272728|
|19|91|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|20|93|0.000000|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0.04|
|21|99|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0.12333333333333332|
|22|100|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0|
|23|101|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/names_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/names_test.go)|0.022727272727272728|
|24|102|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/arg_placeholder_with_call_model.py|0.5|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/call_with_call_model.py|0.5|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/ggnn_subtoken.py|0.5|
|kite-go/lang/python/pythoncomplete/offline/cmds/recalltest/main.go|0.12333333333333332|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/attr.py|0.1|
|kite-go/lang/python/pythoncomplete/performancetest/performance_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/call_patterns_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/dict_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/imports_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/keywords_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/kwargs_test.go|0.022727272727272728|

## [`kite-go/lang/python/pythondocs/index.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/index.go)

Coverage: 1.015656

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-golib/tfidf/termcounter_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tfidf/termcounter_test.go)|0|
|1|6|0.015625|[`kite-golib/text/tokenizer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/text/tokenizer_test.go)|0|
|2|15|0.000031|[`kite-golib/tfidf/scorer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tfidf/scorer_test.go)|0|
|3|39|0.000000|[`kite-go/lang/python/pythondocs/response_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/response_test.go)|0|
|4|52|0.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|0|
|5|60|0.000000|[`kite-go/lang/python/pythongraph/traindata/stringops_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph/traindata/stringops_test.go)|0|
|6|64|0.000000|[`kite-go/lang/python/pythonranker/featurers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonranker/featurers_test.go)|0|
|7|87|0.000000|[`kite-go/lang/python/pythondocs/testutils_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/testutils_test.go)|0|
|8|89|0.000000|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|0|
|9|96|0.000000|[`kite-go/lang/python/pythoncode/stats_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncode/stats_test.go)|0|
|10|108|0.000000|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0|
|11|122|0.000000|[`kite-go/lang/python/pythoncomplete/offline/cmds/performancetest/histogram.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/offline/cmds/performancetest/histogram.go)|0|
|12|128|0.000000|[`kite-go/client/internal/kitelocal/event_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/event_processor_test.go)|0|
|13|129|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|14|147|0.000000|[`kite-go/lang/python/pythonindex/client_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonindex/client_test.go)|0|
|15|152|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/completions_slow_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/completions_slow_test.go)|0|
|16|157|0.000000|[`kite-go/lang/python/testcorpus/testcorpus.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/testcorpus/testcorpus.go)|0|
|17|161|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/completions_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/completions_test.go)|0|
|18|171|0.000000|[`kite-golib/pipeline/engine_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/pipeline/engine_test.go)|0|
|19|173|0.000000|[`kite-golib/lexicalv0/cmds/test_inference_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_inference_latency/main.go)|0|
|20|175|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0|
|21|181|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0|
|22|186|0.000000|[`kite-golib/lexicalv0/javascript/treesitter_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript/treesitter_test.go)|0|
|23|190|0.000000|[`kite-go/clientlogs/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs/server_test.go)|0|
|24|192|0.000000|[`kite-golib/complete/corpustests/testcase.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/corpustests/testcase.go)|0|

# Files using text only

## [`kite-go/navigation/recommend/recommend.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend.go)

Coverage: 1.250149

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/navigation/recommend/recommend_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend_test.go)|0.05263157894736842|
|1|2|0.250000|[`kite-go/navigation/recommend/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/load_test.go)|0.05263157894736842|
|2|13|0.000122|[`kite-go/navigation/offline/validation/validation_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/validation_test.go)|0.05263157894736842|
|3|16|0.000015|[`kite-go/navigation/recommend/vectorizer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/vectorizer_test.go)|0.05263157894736842|
|4|17|0.000008|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|0.05263157894736842|
|5|18|0.000004|[`kite-go/navigation/recommend/graph_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/graph_test.go)|0.05263157894736842|
|6|21|0.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|0.05263157894736842|
|7|24|0.000000|[`kite-go/navigation/git/git_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/git_test.go)|0|
|8|26|0.000000|[`kite-go/navigation/codebase/project_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/project_test.go)|0.05263157894736842|
|9|27|0.000000|[`kite-go/navigation/metrics/metrics_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/metrics/metrics_test.go)|0|
|10|29|0.000000|[`kite-go/navigation/ignore/ignore_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/ignore/ignore_test.go)|0|
|11|32|0.000000|[`kite-go/navigation/localpath/localpath_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/localpath/localpath_test.go)|0|
|12|50|0.000000|[`kite-go/navigation/recommend/files_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/files_test.go)|0.05263157894736842|
|13|54|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0.05263157894736842|
|14|83|0.000000|[`kite-go/client/internal/kitelocal/internal/indexing/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/indexing/manager_test.go)|0|
|15|84|0.000000|[`kite-go/client/internal/kitelocal/internal/filesystem/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/filesystem/manager_test.go)|0|
|16|122|0.000000|[`kite-go/navigation/git/storage_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/storage_test.go)|0|
|17|132|0.000000|[`local-pipelines/mixing/data/normalize/analyze_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/mixing/data/normalize/analyze_test.go)|0|
|18|144|0.000000|[`kite-go/client/internal/kitelocal/test/remotecontrol.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/remotecontrol.go)|0|
|19|161|0.000000|[`kite-golib/lexicalv0/text/split_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/split_test.go)|0|
|20|169|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0|
|21|173|0.000000|[`kite-go/client/internal/localpath/localpath_unixlike_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/localpath/localpath_unixlike_test.go)|0|
|22|193|0.000000|[`kite-go/localfiles/file_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/localfiles/file_test.go)|0|
|23|210|0.000000|[`kite-go/lang/python/pythonbatch/selectfiles_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonbatch/selectfiles_test.go)|0|
|24|211|0.000000|[`kite-go/hmacutil/hmac_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/hmacutil/hmac_test.go)|0|

## [`kite-go/client/internal/kitelocal/internal/navigation/manager.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager.go)

Coverage: 1.257816

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/client/internal/kitelocal/internal/navigation/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/navigation/manager_test.go)|15.134642160957952|
|1|2|0.250000|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|3.3065998329156225|
|2|7|0.007812|[`kite-go/navigation/recommend/recommend_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/recommend_test.go)|0.05263157894736842|
|3|18|0.000004|[`kite-go/navigation/codebase/project_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/project_test.go)|1.556599832915622|
|4|28|0.000000|[`kite-go/client/internal/status/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/status/manager_test.go)|0.18518518518518517|
|5|52|0.000000|[`kite-go/client/internal/metrics/livemetrics/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/manager_test.go)|0.018518518518518517|
|6|61|0.000000|[`kite-go/client/internal/ws/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/ws/manager_test.go)|0.1111111111111111|
|7|62|0.000000|[`kite-go/client/internal/autocorrect/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autocorrect/manager_test.go)|0|
|8|71|0.000000|[`kite-go/client/internal/clientapp/test/mainloop_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test/mainloop_test.go)|0|
|9|74|0.000000|[`kite-go/navigation/git/git_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/git/git_test.go)|0.08333333333333333|
|10|84|0.000000|[`kite-go/client/internal/desktoplogin/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/desktoplogin/manager_test.go)|0|
|11|85|0.000000|[`kite-go/client/internal/kitelocal/test/indexing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/indexing_test.go)|0|
|12|87|0.000000|[`kite-go/client/internal/kitelocal/test/autosearch.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/test/autosearch.go)|0|
|13|88|0.000000|[`kite-go/client/internal/settings/compat_manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/compat_manager_test.go)|0.35185185185185186|
|14|89|0.000000|[`kite-golib/wstest/helpers.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/wstest/helpers.go)|0.1111111111111111|
|15|92|0.000000|[`kite-go/navigation/recommend/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/recommend/load_test.go)|0.05263157894736842|
|16|97|0.000000|[`kite-go/client/internal/kitelocal/event_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/event_processor_test.go)|0|
|17|104|0.000000|[`kite-go/client/internal/clientapp/test_setup.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test_setup.go)|0|
|18|111|0.000000|[`kite-go/client/internal/settings/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings/manager_test.go)|0|
|19|116|0.000000|[`kite-go/client/internal/autosearch/test/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autosearch/test/manager_test.go)|0|
|20|118|0.000000|[`kite-go/navigation/offline/validation/validation_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/validation_test.go)|0.05263157894736842|
|21|125|0.000000|[`kite-go/clientlogs/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs/server_test.go)|0|
|22|126|0.000000|[`kite-go/client/internal/kitelocal/permissions/handlers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/handlers_test.go)|0|
|23|131|0.000000|[`kite-go/client/internal/clientapp/test/panic_component.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/test/panic_component.go)|0|
|24|134|0.000000|[`kite-go/client/internal/auth/proxy_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/auth/proxy_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/navigation/offline/validation/load_test.go|0.05263157894736842|
|kite-go/navigation/recommend/files_test.go|0.05263157894736842|
|kite-go/navigation/recommend/graph_test.go|0.05263157894736842|
|kite-go/navigation/recommend/vectorizer_test.go|0.05263157894736842|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go|0.14285714285714285|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go|0.14285714285714285|
|kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go|0.14285714285714285|
|kite-go/navigation/ignore/ignore_test.go|0.25|
|kite-go/navigation/ignore/munge_test.go|0.25|
|kite-go/client/internal/conversion/cohort/autostart-trial_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/cta_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/quiet-autostart_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/usage-paywall_test.go|0.018518518518518517|
|kite-go/client/internal/kitelocal/internal/signatures/manager_test.go|0.018518518518518517|
|kite-go/client/internal/metrics/proselected_test.go|0.018518518518518517|
|kite-go/client/internal/notifications/manager_test.go|0.018518518518518517|
|local-pipelines/lexical/train/cmds/utils/datasets_test.go|0.018518518518518517|
|kite-go/navigation/git/cache_test.go|0.08333333333333333|
|kite-go/navigation/git/storage_test.go|0.08333333333333333|

## [`kite-go/lang/language.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/language.go)

Coverage: 1.503540

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-go/lang/language_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/language_test.go)|1.0|
|1|1|0.500000|[`kite-golib/lexicalv0/langgroup_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/langgroup_test.go)|0|
|2|9|0.001953|[`kite-go/client/internal/metrics/livemetrics/languagemetrics_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/languagemetrics_test.go)|0|
|3|10|0.000977|[`local-pipelines/lexical/train/cmds/utils/datasets_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/lexical/train/cmds/utils/datasets_test.go)|0.8333333333333333|
|4|11|0.000488|[`kite-go/client/internal/kitelocal/permissions/support_status_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/support_status_test.go)|1.0|
|5|13|0.000122|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|6|27|0.000000|[`kite-go/navigation/localpath/localpath_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/localpath/localpath_test.go)|0|
|7|35|0.000000|[`kite-go/client/internal/clientapp/lang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/lang_test.go)|0|
|8|49|0.000000|[`kite-golib/lexicalv0/inspect/inspect_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/inspect/inspect_test.go)|0|
|9|52|0.000000|[`kite-golib/lexicalv0/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/context_test.go)|0|
|10|61|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_darwin_test.go)|0|
|12|65|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|13|70|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_linux_test.go)|0|
|14|74|0.000000|[`kite-go/client/internal/kitelocal/permissions/handlers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/handlers_test.go)|0|
|15|89|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0|
|16|96|0.000000|[`kite-golib/lexicalv0/javascript/render_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript/render_test.go)|0|
|17|104|0.000000|[`kite-go/curation/titleparser/utility_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/curation/titleparser/utility_test.go)|0|
|18|105|0.000000|[`kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new/internal/vscode/vscode_windows_test.go)|0|
|19|112|0.000000|[`kite-go/lang/detect_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/detect_test.go)|0|
|20|131|0.000000|[`kite-go/localfiles/server_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/localfiles/server_test.go)|0|
|21|140|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0|
|22|147|0.000000|[`kite-golib/lexicalv0/words/count_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/words/count_test.go)|0|
|23|176|0.000000|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0|
|24|181|0.000000|[`kite-golib/lexicalv0/javascript/javascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript/javascript_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-golib/lexicalv0/encoder_test.go|0.5|
|kite-golib/lexicalv0/text/split_test.go|0.29166666666666663|
|kite-golib/lexicalv0/text/lexer_test.go|0.125|
|kite-golib/lexicalv0/text/render_test.go|0.125|

## [`kite-go/client/internal/kitelocal/internal/completions/lexical.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/completions/lexical.go)

Coverage: 0.003052

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|9|0.001953|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|1|10|0.000977|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0|
|2|13|0.000122|[`kite-golib/complete/data/api_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/api_test.go)|0|
|3|21|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0|
|4|25|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0|
|5|26|0.000000|[`kite-go/client/internal/metrics/completions/metric_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/completions/metric_test.go)|0|
|6|29|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0|
|7|33|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|0|
|8|34|0.000000|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0|
|9|39|0.000000|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|10|42|0.000000|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0|
|11|56|0.000000|[`kite-go/client/platform/platform_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/platform/platform_test.go)|0|
|12|65|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0|
|13|67|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|14|71|0.000000|[`kite-go/client/internal/notifications/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/notifications/manager_test.go)|0.018518518518518517|
|15|74|0.000000|[`kite-go/userids/userids_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/userids/userids_test.go)|0|
|16|81|0.000000|[`kite-go/lang/python/testcorpus/testcorpus.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/testcorpus/testcorpus.go)|0|
|17|83|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/texthtml_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/texthtml_test.go)|0|
|18|86|0.000000|[`kite-go/client/internal/kitelocal/internal/indexing/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/indexing/manager_test.go)|0|
|19|87|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0|
|20|90|0.000000|[`kite-go/client/internal/metrics/livemetrics/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/livemetrics/manager_test.go)|0.018518518518518517|
|21|92|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0|
|22|93|0.000000|[`kite-golib/complete/data/completion_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/completion_test.go)|0|
|23|99|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textvue_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textvue_test.go)|0|
|24|101|0.000000|[`kite-golib/lexicalv0/text/comment_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text/comment_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/client/internal/settings/compat_manager_test.go|0.11851851851851852|
|kite-go/client/internal/kitelocal/test/editorapi_buffer_test.go|0.14285714285714285|
|kite-go/client/internal/metrics/livemetrics_test/metrics_test.go|0.14285714285714285|
|kite-go/client/internal/kitelocal/event_processor_test.go|0.14285714285714285|
|kite-go/client/internal/conversion/cohort/autostart-trial_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/cta_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/quiet-autostart_test.go|0.018518518518518517|
|kite-go/client/internal/conversion/cohort/usage-paywall_test.go|0.018518518518518517|
|kite-go/client/internal/kitelocal/internal/navigation/manager_test.go|0.018518518518518517|
|kite-go/client/internal/kitelocal/internal/signatures/manager_test.go|0.018518518518518517|
|kite-go/client/internal/metrics/proselected_test.go|0.018518518518518517|
|kite-go/client/internal/status/manager_test.go|0.018518518518518517|
|local-pipelines/lexical/train/cmds/utils/datasets_test.go|0.018518518518518517|

## [`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/Data_inputs.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/Data_inputs.go)

Coverage: 0.537295

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|1|0.500000|[`kite-golib/lexicalv0/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/context_test.go)|1.342156862745098|
|1|5|0.031250|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|1.2509803921568627|
|2|8|0.003906|[`kite-golib/lexicalv0/langgroup_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/langgroup_test.go)|0|
|3|9|0.001953|[`local-pipelines/lexical/train/cmds/utils/datasets_test.go`](https://github.com/kiteco/kiteco/blob/master/local-pipelines/lexical/train/cmds/utils/datasets_test.go)|0|
|4|13|0.000122|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0.1801470588235294|
|5|14|0.000061|[`kite-go/client/internal/kitelocal/permissions/support_status_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/permissions/support_status_test.go)|0|
|6|19|0.000002|[`kite-go/client/internal/clientapp/lang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp/lang_test.go)|0|
|7|23|0.000000|[`kite-golib/lexicalv0/predict/context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/predict/context_test.go)|0|
|8|24|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0|
|9|27|0.000000|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0|
|10|29|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0|
|11|32|0.000000|[`kite-golib/lexicalv0/cmds/test_search_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_search_latency/main.go)|0.24136321195144722|
|12|42|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0.3333333333333333|
|13|44|0.000000|[`kite-golib/lexicalv0/cmds/test_inference_latency/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_inference_latency/main.go)|0.13025210084033612|
|14|47|0.000000|[`kite-golib/lexicalv0/tfserving/cmds/test_client/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving/cmds/test_client/main.go)|0.058823529411764705|
|15|53|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0.8333333333333333|
|16|55|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|17|56|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0.6333333333333333|
|18|57|0.000000|[`kite-go/lang/lexical/lexicalcomplete/api/events_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/api/events_test.go)|0|
|19|62|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/curated_context_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/curated_context_test.go)|0.058823529411764705|
|20|64|0.000000|[`kite-go/lang/python/pythoncomplete/performancetest/testenv.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/performancetest/testenv.go)|0|
|21|68|0.000000|[`kite-golib/lexicalv0/encoder_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/encoder_test.go)|0.33723262032085566|
|22|71|0.000000|[`kite-golib/lexicalv0/cmds/test_prettify_golang/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_prettify_golang/main.go)|0|
|23|72|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/utils_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/utils_test.go)|0|
|24|81|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-golib/lexicalv0/javascript/javascript_test.go|0.24632352941176472|
|kite-golib/lexicalv0/javascript/treesitter_test.go|0.24632352941176472|
|kite-golib/lexicalv0/text/split_test.go|0.24632352941176472|
|kite-golib/lexicalv0/inspect/inspect_test.go|0.09090909090909091|
|kite-go/client/internal/metrics/completions/metric_test.go|0.058823529411764705|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/agnostic_test.go|0.058823529411764705|
|kite-golib/lexicalv0/javascript/render_test.go|0.058823529411764705|
|kite-golib/lexicalv0/text/lexer_test.go|0.7588235294117647|
|kite-golib/lexicalv0/cmds/compare_text_model/normalize_test.go|0.058823529411764705|
|kite-go/client/internal/status/manager_test.go|0.058823529411764705|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/autoCloseParen.go|0.5333333333333333|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/noNonMatchedCloseParen.go|0.5333333333333333|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/dedupeCall.go|0.2|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/identifierMiddleOfLine.go|0.2|
|kite-golib/lexicalv0/text/render_test.go|0.2|
|kite-golib/lexicalv0/css/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/html/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/python/lexer_test.go|0.1213235294117647|
|kite-golib/lexicalv0/vue/lexer_test.go|0.1213235294117647|
|kite-go/lang/lexical/lexicalcomplete/corpustest/corpus/text/basic1.js|0.1|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textc_test.go|0.43333333333333335|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/texthtml_test.go|0.1|
|kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjava_test.go|0.1|
|kite-golib/lexicalv0/predict/partialrun_test.go|0.1|

## [`kite-go/lang/python/pythoncomplete/driver/mixing.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing.go)

Coverage: 0.015658

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|6|0.015625|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0.022727272727272728|
|1|15|0.000031|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0.4793939393939394|
|2|19|0.000002|[`kite-go/navigation/offline/validation/load_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/offline/validation/load_test.go)|0|
|3|22|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/call_test.go)|0.022727272727272728|
|4|23|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/lexical_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/lexical_test.go)|0.022727272727272728|
|5|24|0.000000|[`kite-go/lang/python/pythoncomplete/api/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/api/mixing_test.go)|0.14606060606060606|
|6|30|0.000000|[`kite-golib/complete/corpustests/testcase.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/corpustests/testcase.go)|0.6|
|7|32|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/providers_test.go)|0.022727272727272728|
|8|44|0.000000|[`kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/recalltest/recallcomputer.go)|0.12333333333333332|
|9|45|0.000000|[`kite-golib/complete/data/completion_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/completion_test.go)|0|
|10|46|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/empty_calls_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/empty_calls_test.go)|0.022727272727272728|
|11|50|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/providers_test.go)|0|
|12|53|0.000000|[`kite-go/lang/python/pythoncomplete/performancetest/testenv.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/performancetest/testenv.go)|0.14606060606060606|
|13|63|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/attributes_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/attributes_test.go)|0.022727272727272728|
|14|69|0.000000|[`kite-go/client/internal/metrics/completions/metric_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics/completions/metric_test.go)|0|
|15|72|0.000000|[`kite-golib/complete/data/api_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/data/api_test.go)|0|
|16|75|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textpython_test.go)|0|
|17|80|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/ggnn_provider_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/ggnn_provider_test.go)|0.022727272727272728|
|18|84|0.000000|[`kite-golib/lexicalv0/cmds/test_ast_based_render/main.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds/test_ast_based_render/main.go)|0.04|
|19|86|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textjavascript_test.go)|0|
|20|89|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/names_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/names_test.go)|0.022727272727272728|
|21|90|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textgolang_test.go)|0|
|22|97|0.000000|[`kite-go/lang/python/pythoncomplete/pythonproviders/imports_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/pythonproviders/imports_test.go)|0.022727272727272728|
|23|99|0.000000|[`kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textvue_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/lexicalproviders/textvue_test.go)|0|
|24|108|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0.12333333333333332|

Relevant but not retrieved:

|Test|Weighted Hits|
|-|-|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/lexical.py|0.8333333333333333|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/arg_placeholder_with_call_model.py|0.5|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/call_with_call_model.py|0.5|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/ggnn_subtoken.py|0.5|
|kite-go/lang/python/pythoncomplete/offline/cmds/recalltest/main.go|0.12333333333333332|
|kite-go/lang/python/pythoncomplete/corpustest/corpus/attr.py|0.1|
|kite-go/client/internal/settings/compat_manager_test.go|0.2|
|kite-go/lang/python/pythoncomplete/performancetest/performance_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/call_patterns_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/dict_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/keywords_test.go|0.022727272727272728|
|kite-go/lang/python/pythoncomplete/pythonproviders/kwargs_test.go|0.022727272727272728|

## [`kite-go/lang/python/pythondocs/index.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/index.go)

Coverage: 1.015686

Retrieved:

|Test rank|Total rank|Coverage|Test|Weighted Hits|
|-|-|-|-|-|
|0|0|1.000000|[`kite-golib/tfidf/termcounter_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tfidf/termcounter_test.go)|0|
|1|6|0.015625|[`kite-golib/text/tokenizer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/text/tokenizer_test.go)|0|
|2|14|0.000061|[`kite-golib/tfidf/scorer_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tfidf/scorer_test.go)|0|
|3|33|0.000000|[`kite-go/lang/python/pythondocs/response_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/response_test.go)|0|
|4|48|0.000000|[`kite-go/lang/python/pythongraph/traindata/stringops_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph/traindata/stringops_test.go)|0|
|5|52|0.000000|[`kite-go/lang/python/pythonranker/featurers_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonranker/featurers_test.go)|0|
|6|67|0.000000|[`kite-go/lang/python/pythondocs/testutils_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs/testutils_test.go)|0|
|7|73|0.000000|[`kite-go/lang/python/pythoncode/stats_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncode/stats_test.go)|0|
|8|90|0.000000|[`kite-go/lang/python/pythoncomplete/offline/cmds/performancetest/histogram.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/offline/cmds/performancetest/histogram.go)|0|
|9|95|0.000000|[`kite-go/lang/python/pythoncomplete/driver/mixing_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/driver/mixing_test.go)|0|
|10|112|0.000000|[`kite-go/lang/python/pythonindex/client_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonindex/client_test.go)|0|
|11|116|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/completions_slow_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/completions_slow_test.go)|0|
|12|118|0.000000|[`kite-go/client/internal/kitelocal/event_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/event_processor_test.go)|0|
|13|121|0.000000|[`kite-go/lang/python/testcorpus/testcorpus.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/testcorpus/testcorpus.go)|0|
|14|125|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/completions_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/completions_test.go)|0|
|15|134|0.000000|[`kite-golib/pipeline/engine_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/pipeline/engine_test.go)|0|
|16|149|0.000000|[`kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/lexical/lexicalcomplete/corpustest/shared_test.go)|0|
|17|153|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/completions_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/completions_test.go)|0|
|18|159|0.000000|[`kite-golib/lexicalv0/javascript/treesitter_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript/treesitter_test.go)|0|
|19|187|0.000000|[`kite-go/lang/python/pythoncomplete/corpustest/shared_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete/corpustest/shared_test.go)|0|
|20|188|0.000000|[`kite-go/client/internal/autosearch/test/manager_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autosearch/test/manager_test.go)|0|
|21|215|0.000000|[`kite-golib/complete/corpustests/testcase.go`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete/corpustests/testcase.go)|0|
|22|216|0.000000|[`kite-go/client/internal/kitelocal/file_processor_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/file_processor_test.go)|0|
|23|217|0.000000|[`kite-go/navigation/codebase/codebase_test.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation/codebase/codebase_test.go)|0|
|24|231|0.000000|[`kite-go/lang/python/pythonkeyword/cmds/model-test/examples.go`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonkeyword/cmds/model-test/examples.go)|0|


# Directories

|Directory|Coverage|Number of files|
|-|-|-|
|[`services`](https://github.com/kiteco/kiteco/blob/master/services)|0.781799|2|
|[`lambda-functions`](https://github.com/kiteco/kiteco/blob/master/lambda-functions)|0.535367|14|
|[`kite-go`](https://github.com/kiteco/kiteco/blob/master/kite-go)|0.252564|1397|
|[`kite-golib`](https://github.com/kiteco/kiteco/blob/master/kite-golib)|0.247924|329|
|[`kite-python`](https://github.com/kiteco/kiteco/blob/master/kite-python)|0.148939|197|
|[`linux`](https://github.com/kiteco/kiteco/blob/master/linux)|0.124030|22|
|[`kite-server`](https://github.com/kiteco/kiteco/blob/master/kite-server)|0.091084|19|
|[`windows`](https://github.com/kiteco/kiteco/blob/master/windows)|0.090834|57|
|[`kite-answers`](https://github.com/kiteco/kiteco/blob/master/kite-answers)|0.087773|71|
|[`devops`](https://github.com/kiteco/kiteco/blob/master/devops)|0.086393|11|
|[`sidebar`](https://github.com/kiteco/kiteco/blob/master/sidebar)|0.075525|176|
|[`kite-exp`](https://github.com/kiteco/kiteco/blob/master/kite-exp)|0.075480|77|
|[`local-pipelines`](https://github.com/kiteco/kiteco/blob/master/local-pipelines)|0.050823|157|
|[`emr-pipelines`](https://github.com/kiteco/kiteco/blob/master/emr-pipelines)|0.045220|47|
|[`web`](https://github.com/kiteco/kiteco/blob/master/web)|0.026438|196|
|[`docker-images`](https://github.com/kiteco/kiteco/blob/master/docker-images)|0.022084|7|
|[`osx`](https://github.com/kiteco/kiteco/blob/master/osx)|0.017735|18|
|[`airflow`](https://github.com/kiteco/kiteco/blob/master/airflow)|0.012854|25|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-go/client/internal/mockserver`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/mockserver)|1.023488|6|
|[`kite-go/client/internal/conversion`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/conversion)|0.817623|8|
|[`kite-go/client/internal/status`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/status)|0.816956|2|
|[`kite-go/client/internal/metrics`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/metrics)|0.669142|19|
|[`kite-go/client/internal/settings`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/settings)|0.639194|4|
|[`kite-go/client/internal/clienttelemetry`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clienttelemetry)|0.523483|3|
|[`kite-go/client/internal/proxy`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/proxy)|0.507822|2|
|[`kite-go/client/internal/notifications`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/notifications)|0.464869|3|
|[`kite-go/client/internal/kitelocal`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal)|0.457537|39|
|[`kite-go/client/internal/localpath`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/localpath)|0.407123|3|
|[`kite-go/client/internal/plugins_new`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins_new)|0.380803|87|
|[`kite-go/client/internal/plugins`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/plugins)|0.321167|2|
|[`kite-go/client/internal/statusicon`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/statusicon)|0.304695|5|
|[`kite-go/client/internal/systeminfo`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/systeminfo)|0.285159|2|
|[`kite-go/client/internal/client`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/client)|0.272959|5|
|[`kite-go/client/internal/performance`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/performance)|0.265837|11|
|[`kite-go/client/internal/watch`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/watch)|0.260149|4|
|[`kite-go/client/internal/autocorrect`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/autocorrect)|0.250080|2|
|[`kite-go/client/internal/auth`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/auth)|0.219902|11|
|[`kite-go/client/internal/desktoplogin`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/desktoplogin)|0.082050|2|
|[`kite-go/client/internal/windowsui`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/windowsui)|0.044427|2|
|[`kite-go/client/internal/clientapp`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/clientapp)|0.040624|4|
|[`kite-go/client/internal/reg`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/reg)|0.007813|2|
|[`kite-go/client/internal/health`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/health)|0.005412|3|
|[`kite-go/client/internal/updates`](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/updates)|0.000554|8|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-go/clientlogs`](https://github.com/kiteco/kiteco/blob/master/kite-go/clientlogs)|1.265778|2|
|[`kite-go/clustering`](https://github.com/kiteco/kiteco/blob/master/kite-go/clustering)|0.854350|3|
|[`kite-go/conversion`](https://github.com/kiteco/kiteco/blob/master/kite-go/conversion)|0.836716|3|
|[`kite-go/navigation`](https://github.com/kiteco/kiteco/blob/master/kite-go/navigation)|0.781080|27|
|[`kite-go/annotate`](https://github.com/kiteco/kiteco/blob/master/kite-go/annotate)|0.729371|8|
|[`kite-go/typeinduction`](https://github.com/kiteco/kiteco/blob/master/kite-go/typeinduction)|0.638137|5|
|[`kite-go/health`](https://github.com/kiteco/kiteco/blob/master/kite-go/health)|0.628915|4|
|[`kite-go/sandbox`](https://github.com/kiteco/kiteco/blob/master/kite-go/sandbox)|0.585508|12|
|[`kite-go/release`](https://github.com/kiteco/kiteco/blob/master/kite-go/release)|0.519596|4|
|[`kite-go/localfiles`](https://github.com/kiteco/kiteco/blob/master/kite-go/localfiles)|0.515437|11|
|[`kite-go/event`](https://github.com/kiteco/kiteco/blob/master/kite-go/event)|0.449251|9|
|[`kite-go/client`](https://github.com/kiteco/kiteco/blob/master/kite-go/client)|0.383216|328|
|[`kite-go/diff`](https://github.com/kiteco/kiteco/blob/master/kite-go/diff)|0.376355|4|
|[`kite-go/websandbox`](https://github.com/kiteco/kiteco/blob/master/kite-go/websandbox)|0.347843|4|
|[`kite-go/ranking`](https://github.com/kiteco/kiteco/blob/master/kite-go/ranking)|0.278722|9|
|[`kite-go/community`](https://github.com/kiteco/kiteco/blob/master/kite-go/community)|0.255068|33|
|[`kite-go/curation`](https://github.com/kiteco/kiteco/blob/master/kite-go/curation)|0.225194|59|
|[`kite-go/lang`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang)|0.210340|592|
|[`kite-go/web`](https://github.com/kiteco/kiteco/blob/master/kite-go/web)|0.149340|8|
|[`kite-go/autocorrect`](https://github.com/kiteco/kiteco/blob/master/kite-go/autocorrect)|0.125293|5|
|[`kite-go/localcode`](https://github.com/kiteco/kiteco/blob/master/kite-go/localcode)|0.096659|19|
|[`kite-go/response`](https://github.com/kiteco/kiteco/blob/master/kite-go/response)|0.070230|11|
|[`kite-go/cmds`](https://github.com/kiteco/kiteco/blob/master/kite-go/cmds)|0.068191|92|
|[`kite-go/summarize`](https://github.com/kiteco/kiteco/blob/master/kite-go/summarize)|0.038951|13|
|[`kite-go/dynamicanalysis`](https://github.com/kiteco/kiteco/blob/master/kite-go/dynamicanalysis)|0.031875|17|
|[`kite-go/stackoverflow`](https://github.com/kiteco/kiteco/blob/master/kite-go/stackoverflow)|0.028911|35|
|[`kite-go/lsp`](https://github.com/kiteco/kiteco/blob/master/kite-go/lsp)|0.027691|24|
|[`kite-go/knowledge`](https://github.com/kiteco/kiteco/blob/master/kite-go/knowledge)|0.025372|12|
|[`kite-go/core`](https://github.com/kiteco/kiteco/blob/master/kite-go/core)|0.021851|2|
|[`kite-go/codewrap`](https://github.com/kiteco/kiteco/blob/master/kite-go/codewrap)|0.021760|4|
|[`kite-go/segment-analysis`](https://github.com/kiteco/kiteco/blob/master/kite-go/segment-analysis)|0.021467|17|
|[`kite-go/github`](https://github.com/kiteco/kiteco/blob/master/kite-go/github)|0.014393|9|
|[`kite-go/traindata`](https://github.com/kiteco/kiteco/blob/master/kite-go/traindata)|0.002604|3|
|[`kite-go/deployments`](https://github.com/kiteco/kiteco/blob/master/kite-go/deployments)|0.000000|3|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-go/lang/python/pythonenv`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonenv)|1.132167|3|
|[`kite-go/lang/python/pythonkeyword`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonkeyword)|0.696110|5|
|[`kite-go/lang/python/pythonstatic`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonstatic)|0.507481|16|
|[`kite-go/lang/python/pythonimports`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonimports)|0.506326|14|
|[`kite-go/lang/python/pythonscanner`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonscanner)|0.456195|8|
|[`kite-go/lang/python/pythonmetrics`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonmetrics)|0.335950|3|
|[`kite-go/lang/python/pythonlocal`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonlocal)|0.332326|5|
|[`kite-go/lang/python/pythonparser`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonparser)|0.331911|35|
|[`kite-go/lang/python/pythondocs`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythondocs)|0.312556|24|
|[`kite-go/lang/python/pythonbatch`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonbatch)|0.310246|12|
|[`kite-go/lang/python/pythonast`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonast)|0.291074|10|
|[`kite-go/lang/python/pythonindex`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonindex)|0.280725|12|
|[`kite-go/lang/python/seo`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/seo)|0.254202|5|
|[`kite-go/lang/python/pythonranker`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonranker)|0.249681|18|
|[`kite-go/lang/python/pythonresource`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonresource)|0.202096|69|
|[`kite-go/lang/python/pythonskeletons`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonskeletons)|0.193285|8|
|[`kite-go/lang/python/pythonexpr`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonexpr)|0.171165|11|
|[`kite-go/lang/python/pythoncode`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncode)|0.154524|13|
|[`kite-go/lang/python/pythoncomplete`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncomplete)|0.131646|49|
|[`kite-go/lang/python/cmds`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/cmds)|0.123228|31|
|[`kite-go/lang/python/pythonautocorrect`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonautocorrect)|0.119537|12|
|[`kite-go/lang/python/pythongraph`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythongraph)|0.102443|57|
|[`kite-go/lang/python/pythonpipeline`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonpipeline)|0.101975|6|
|[`kite-go/lang/python/pythonanalyzer`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonanalyzer)|0.098969|2|
|[`kite-go/lang/python/pythontype`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythontype)|0.068941|33|
|[`kite-go/lang/python/pythoncuration`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythoncuration)|0.062157|9|
|[`kite-go/lang/python/pythontracking`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythontracking)|0.014985|19|
|[`kite-go/lang/python/pythonmodels`](https://github.com/kiteco/kiteco/blob/master/kite-go/lang/python/pythonmodels)|0.000601|14|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-golib/conversion`](https://github.com/kiteco/kiteco/blob/master/kite-golib/conversion)|0.891991|2|
|[`kite-golib/rollbar`](https://github.com/kiteco/kiteco/blob/master/kite-golib/rollbar)|0.875567|2|
|[`kite-golib/decisiontree`](https://github.com/kiteco/kiteco/blob/master/kite-golib/decisiontree)|0.774430|2|
|[`kite-golib/languagemodel`](https://github.com/kiteco/kiteco/blob/master/kite-golib/languagemodel)|0.718792|3|
|[`kite-golib/tfidf`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tfidf)|0.625992|3|
|[`kite-golib/telemetry`](https://github.com/kiteco/kiteco/blob/master/kite-golib/telemetry)|0.625203|6|
|[`kite-golib/text`](https://github.com/kiteco/kiteco/blob/master/kite-golib/text)|0.552075|5|
|[`kite-golib/stringindex`](https://github.com/kiteco/kiteco/blob/master/kite-golib/stringindex)|0.511810|2|
|[`kite-golib/linenumber`](https://github.com/kiteco/kiteco/blob/master/kite-golib/linenumber)|0.511787|2|
|[`kite-golib/errors`](https://github.com/kiteco/kiteco/blob/master/kite-golib/errors)|0.500496|2|
|[`kite-golib/licensing`](https://github.com/kiteco/kiteco/blob/master/kite-golib/licensing)|0.475320|9|
|[`kite-golib/awsutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/awsutil)|0.433799|6|
|[`kite-golib/email`](https://github.com/kiteco/kiteco/blob/master/kite-golib/email)|0.354269|3|
|[`kite-golib/kitectx`](https://github.com/kiteco/kiteco/blob/master/kite-golib/kitectx)|0.339569|5|
|[`kite-golib/lexicalv0`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0)|0.301726|102|
|[`kite-golib/diskmap`](https://github.com/kiteco/kiteco/blob/master/kite-golib/diskmap)|0.258724|7|
|[`kite-golib/fileutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/fileutil)|0.258128|6|
|[`kite-golib/tensorflow`](https://github.com/kiteco/kiteco/blob/master/kite-golib/tensorflow)|0.183737|7|
|[`kite-golib/status`](https://github.com/kiteco/kiteco/blob/master/kite-golib/status)|0.170201|7|
|[`kite-golib/complete`](https://github.com/kiteco/kiteco/blob/master/kite-golib/complete)|0.143588|8|
|[`kite-golib/zseek`](https://github.com/kiteco/kiteco/blob/master/kite-golib/zseek)|0.125153|2|
|[`kite-golib/readdirchanges`](https://github.com/kiteco/kiteco/blob/master/kite-golib/readdirchanges)|0.071810|2|
|[`kite-golib/envutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/envutil)|0.053140|3|
|[`kite-golib/throttle`](https://github.com/kiteco/kiteco/blob/master/kite-golib/throttle)|0.035156|3|
|[`kite-golib/pipeline`](https://github.com/kiteco/kiteco/blob/master/kite-golib/pipeline)|0.018319|48|
|[`kite-golib/segment`](https://github.com/kiteco/kiteco/blob/master/kite-golib/segment)|0.015656|4|
|[`kite-golib/serialization`](https://github.com/kiteco/kiteco/blob/master/kite-golib/serialization)|0.009802|2|
|[`kite-golib/scalinggroups`](https://github.com/kiteco/kiteco/blob/master/kite-golib/scalinggroups)|0.007820|2|
|[`kite-golib/diskmapindex`](https://github.com/kiteco/kiteco/blob/master/kite-golib/diskmapindex)|0.005457|4|
|[`kite-golib/azureutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/azureutil)|0.001100|2|
|[`kite-golib/macaddr`](https://github.com/kiteco/kiteco/blob/master/kite-golib/macaddr)|0.000344|3|
|[`kite-golib/exec`](https://github.com/kiteco/kiteco/blob/master/kite-golib/exec)|0.000245|3|
|[`kite-golib/mixpanel`](https://github.com/kiteco/kiteco/blob/master/kite-golib/mixpanel)|0.000245|4|
|[`kite-golib/stripe`](https://github.com/kiteco/kiteco/blob/master/kite-golib/stripe)|0.000143|4|
|[`kite-golib/octobat`](https://github.com/kiteco/kiteco/blob/master/kite-golib/octobat)|0.000131|2|
|[`kite-golib/bufutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/bufutil)|0.000122|2|
|[`kite-golib/contextutil`](https://github.com/kiteco/kiteco/blob/master/kite-golib/contextutil)|0.000092|3|
|[`kite-golib/codesearch`](https://github.com/kiteco/kiteco/blob/master/kite-golib/codesearch)|0.000047|2|
|[`kite-golib/githubcorpus`](https://github.com/kiteco/kiteco/blob/master/kite-golib/githubcorpus)|0.000040|9|
|[`kite-golib/kitelog`](https://github.com/kiteco/kiteco/blob/master/kite-golib/kitelog)|0.000000|2|
|[`kite-golib/systray`](https://github.com/kiteco/kiteco/blob/master/kite-golib/systray)|0.000000|5|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-golib/lexicalv0/benchmark`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/benchmark)|0.669481|3|
|[`kite-golib/lexicalv0/text`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/text)|0.634089|7|
|[`kite-golib/lexicalv0/render`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/render)|0.607932|3|
|[`kite-golib/lexicalv0/css`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/css)|0.516393|2|
|[`kite-golib/lexicalv0/tfserving`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/tfserving)|0.456750|6|
|[`kite-golib/lexicalv0/bpe`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/bpe)|0.397972|4|
|[`kite-golib/lexicalv0/vue`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/vue)|0.377475|2|
|[`kite-golib/lexicalv0/javascript`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/javascript)|0.286045|5|
|[`kite-golib/lexicalv0/words`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/words)|0.281619|3|
|[`kite-golib/lexicalv0/html`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/html)|0.267169|2|
|[`kite-golib/lexicalv0/predict`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/predict)|0.262945|20|
|[`kite-golib/lexicalv0/inspect`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/inspect)|0.181743|5|
|[`kite-golib/lexicalv0/cmds`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/cmds)|0.140885|8|
|[`kite-golib/lexicalv0/golang`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/golang)|0.114404|5|
|[`kite-golib/lexicalv0/githubdata`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/githubdata)|0.106771|3|
|[`kite-golib/lexicalv0/python`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/python)|0.070711|5|
|[`kite-golib/lexicalv0/localtraining`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/localtraining)|0.028572|5|
|[`kite-golib/lexicalv0/lexer`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/lexer)|0.019713|4|
|[`kite-golib/lexicalv0/performance`](https://github.com/kiteco/kiteco/blob/master/kite-golib/lexicalv0/performance)|0.000733|4|

|Directory|Coverage|Number of files|
|-|-|-|
|[`kite-python/metrics`](https://github.com/kiteco/kiteco/blob/master/kite-python/metrics)|0.696610|4|
|[`kite-python/analysis`](https://github.com/kiteco/kiteco/blob/master/kite-python/analysis)|0.341362|6|
|[`kite-python/kite_emr`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_emr)|0.291666|8|
|[`kite-python/kite_pkgexploration`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_pkgexploration)|0.269131|24|
|[`kite-python/kite_common`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_common)|0.147060|61|
|[`kite-python/kite_ml`](https://github.com/kiteco/kiteco/blob/master/kite-python/kite_ml)|0.076224|87|
|[`kite-python/slackbuildbot`](https://github.com/kiteco/kiteco/blob/master/kite-python/slackbuildbot)|0.023247|4|
|[`kite-python/instrumental`](https://github.com/kiteco/kiteco/blob/master/kite-python/instrumental)|0.008318|2|