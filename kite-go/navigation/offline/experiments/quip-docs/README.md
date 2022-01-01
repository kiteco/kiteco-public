# Experiment description

## Goal

We want to see if a code nav recommender can map code to documentation.
Here we focus on mapping to related Quip documents.

## Data

We collect links from the body of GitHub issues and pull requests.
We are interested in links that point to Quip documents or other GitHub issues or pull requests.
We consider an undirected graph where the nodes are Quip documents, GitHub issues, and pull requests.
An edge indicates there is a link in either direction.
We consider a pull request to be related to a Quip document if they are in the same component of the graph.

## Mapping

We build a code nav recommender on the kiteco repo.
We select options for the code nav recommender so that it does not use git commit data.
We create python files with a Quip document in a string, so the recommender can recommend Quip documents.

## Results

We look at some specific examples based on the collected links.
For each code file, we use the recommender to find related files in the repo.
We discard files that are not one of our collected Quip documents.
We rank the remaining retrieved Quip documents and flag the ones that are relevant.
The results are in `results.md`.

# Instructions to run the experiment

## Set up authentication tokens

Collecting the data from GitHub and Quip requires authentication tokens.
Set up these environment variables before running:
- `GITHUB_AUTH_TOKEN` should contain a token from https://github.com/settings/tokens
- `QUIP_AUTH_TOKEN` should contain a token from https://kite.quip.com/dev/token

## Run

Use `make all` to run the experiment.
