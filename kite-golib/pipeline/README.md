# Description
This package provides a common framework for creating data pipelines that can be used to
analyze properties of a source code corpus, generate training data, etc.

# Interface
- [internal/pipeline/pipeline.go]: for defining a pipeline
- [internal/pipeline/feed.go]: defines the `Feeds` that are the components of the pipeline
- [internal/pipeline/engine.go]: for running the pipeline
