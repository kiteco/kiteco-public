# Summary

Extracts pull request data from GitHub and writes it to disk.

# Options

- Owner: Owner of the repository, e.g. "kiteco" or "apache".
- Repo: Name of the repository, e.g. "kiteco" or "spark".
- WriteDir: Directory to write data to.
- PRState: State of pull requests extracted, "open" or "closed".
- PerPage: Number of pull requests listed per page, e.g. 25.
- NumPulls: Total number of pull requests to extract.
- Comments: Set true to retrieve comments on pull requests.

# Tokens

Uses a personal access token from https://github.com/settings/tokens.

# Schema

- JSON files are serialized structs from `github.com/google/go-github/github`.
- TXT files are certain fields extracted from these structs, intended to be useful for grep.

```
{Options.WriteDir}/
    {PR number}/
        files/
            {Internal file id}/
                diff.json (serialized github.CommitFile)
                patch.txt (github.CommitFile.Patch)
        comments/
            {Internal comment id}/
                body.txt (github.PullRequestComment.Body)
                comment.json (serialized github.PullRequestComment)
                diff.txt (github.PullRequestComment.DiffHunk)
        pull.json (serialized github.PullRequest)
```

# To do:

Create an interface that can be used to read the data, so code doesn't depend on this particular schema.
