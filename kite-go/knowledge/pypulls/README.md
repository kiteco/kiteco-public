The script `pulls.py` gets pull request data from the GitHub API and saves it to a local JSON file.
The JSON file maps pull request numbers to the files in the repo that the pull request edited.
Set up an environment with the dependencies in `requirements.txt` and run the script with two arguments:

```
python pulls.py token repo_name [--months MONTHS]
```

positional arguments:
- `token` is an access token from https://github.com/settings/tokens
- `repo_name` is the full name of the repo, e.g. `vinta/awesome-python`

optional arguments:
  - `MONTHS`  is the number of months back to get pull request data from
