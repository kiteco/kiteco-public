# Experiment description

## Goal

We want to see if a code nav recommender can be used for ranking tests and estimating coverage.

## Ranking tests

We consider some manually selected files and some randomly selected files.
For each file, we use code nav to find related files.
Using a simple heuristic to filter, we list the test files in the recommended order.

To evaluate the quality of the rankings, we look at pairs of code and test files that have been modified by the same commit.
We look at the distribution of the rank of the relevant test files among all test files.
To account for commits that modify many files, we weight the pairs of code and test files by the number of code files modified by the commit.
Note we don't factor in the number of test files in the commit.

## Estimating coverage

For each non-test file, we estimate it's coverage using a simple heuristic.
If many of the top recommended files are test files, we consider the file to be well covered.
For a directory, we estimate the coverage for the directory by averaging over the files in the directory.
We consider some manually selected groups of directories and list their estimated coverage.

## Instructions to run the experiment

Use `make all` to run the experiment.
