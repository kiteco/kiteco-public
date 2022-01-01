# Summarize
This package contains all of the code related to diff summarization.
If you add a new subpackage, please update this README accordingly.

## Package structure

### cmds
This directory contains all binaries related to code summarization. If you would like to add a new binary please place it in this directory. 
We use a single directory structure to make it easier for new developers to find the code they need (a perfectly reasonable alternative would be to have `cmds` subdirectories in each subpackage but for now we opt for a single directory structure so there is no ambiguity about where a binary lives or where to place a new binary).

### data
This package contains all the information relating to the datasets that are used for summarization. This package is the source of truth for which datasets we have, how to iterate over them, and what data structures each dataset exposes.
Plase define any new dataset or type related to diff summarization here.

### encode
This package handles encoding different types of data into a form that can be sent into various models.

### filter
This package handles filtering files.

### model 
This directory contains the summarization models.

### config
This directory defines the configuration parameters used for defining, training, fine tuning, and querying summarization models.


## Related directories

### `kiteco/local-pipelines/summarize`
This is the root directory for "running" summarization pipelines (e.g gather data, extracting data, building vocabs, training models, etc).
This directory should only contain bash scripts and Makefiles related to running summarization pipelines.
Please refrain from placing any exectuable binaries in this directory and instead place them in `./cmds`. The only exception is the python scripts used for training and finetuning models, which live here because putting them under the `kite-go` repo seemed confusing.

### `kiteco/kite-python/kite_ml/kite/summarize`
This directory contains all of the python code that defines the various summarization models.
