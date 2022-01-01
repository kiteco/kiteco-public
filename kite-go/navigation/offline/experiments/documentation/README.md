# Experiment description

## Goal

We want to see if a code nav recommender can map code to documentation.
Here we focus on mapping to dependent `README.md` files.

## Data

We collect some git commits that modified a `README.md` file and a code file.

## Mapping

We build a code nav recommender on the kiteco repo.
We select options for the code nav recommender so that it does not use git commit data.
We create python files with a readmes in a string, so the recommender can recommend readmes.

## Results

We look at some specific examples based on the collected git commits.
We separate the `README.md` files and the code files from the collected commits.
For each code file, we use the recommender to find related files in the repo.
We discard files that are not one of our collected `README.md` files.
We rank the remaining retrieved `README.md` files and flag the ones that are relevant.
The results are in `results.md`.

# Instructions to run the experiment

Use `make all` to run the experiment.
