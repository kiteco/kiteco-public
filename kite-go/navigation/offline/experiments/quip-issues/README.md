# Experiment description

## Goal

We want to see if a code nav recommender can be used for navigating documentation.
Here we focus on mapping to Quip documents to GitHub issues.

## Data

We collect a corpus of GitHub issues and Quip documents.
We collect links from the body of GitHub issues.
We consider an issue to be related to a Quip document if it contains a link to the Quip document.

## Mapping

We build a code nav recommender on the corpus of GitHub issues and Quip documents.
We select options for the code nav recommender so that it does not use git commit data.
We create python files with a document in a string, so the recommender can work with documents.

## Results

We look at some specific examples based on the collected links.
For each Quip document, we use the recommender to find related documents.
We discard files that are not one of our collected GitHub issues.
We rank the remaining retrieved GitHub issues and flag the ones that are relevant.
The results are in `results.md`.
In the histogram, we use a "weighted frequency" so each Quip document has total weight one.
If a document is relevant to `N` issues, then each document-issue pair has weight `1/N`.

# Instructions to run the experiment

## Set up authentication tokens

Collecting the data from GitHub and Quip requires authentication tokens.
Set up these environment variables before running:
- `GITHUB_AUTH_TOKEN` should contain a token from https://github.com/settings/tokens
- `QUIP_AUTH_TOKEN` should contain a token from https://kite.quip.com/dev/token

## Run

Use `make all` to run the experiment.
