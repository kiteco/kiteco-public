By default, all data is loaded from S3 (with a local disk cache) through the use of an internal library, `fileutil`.
Calling `datadeps.Enable()` will configure `fileutil` to automatically load S3 URIs from the filemap bundled into the binary (before falling back to S3),
which can be much faster than loading from S3/disk.

In the main Kited binary, we also call `fileutil.SetLocalOnly()` which configures `fileutil` to never download data from S3, even
if a requested file is not found in the `datadeps`. This prevents released binaries from accessing S3.
