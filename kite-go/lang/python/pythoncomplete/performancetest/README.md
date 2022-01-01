# Completion performance tests

Performance tests are run on `./tests/*.py`.

The marker `$` specifies the offset where completions will be requsted. It will be removed before
the file is passed on to the completion providers.

If you'd like to see more data, e.g. histogram data, then use the cmd `../offline/cmds/performancetest`.

```
cd ../offline/cmds/performancetest
go build .
./performancetest
# or ./performancetest --json output.json if you want detailed data in json format
```  