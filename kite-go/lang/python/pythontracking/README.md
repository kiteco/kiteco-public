# Signature reports

This contains tools to analyze signature logs, which are logged by the usernode and client via Segment and stored in S3.
These tools read the resulting logs from S3 and perform analyses. `internal/context.Recreator` can recreate the context
from a usernode signature failure log, including the resolved AST and local index.
