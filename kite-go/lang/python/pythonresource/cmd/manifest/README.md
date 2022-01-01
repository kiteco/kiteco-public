Kite Resource Manager Manifest Tool
===================================

`manifest` is a utility to make it easier to work with resource data & corresponding manifest files.

```
Usage:
  manifest [command]

Available Commands:
  copydata    copy all files in a manifest to file locations in an isomorphic manifest
  extract     extract a singleton manifest containing only the specified distribution/version
  help        Help about any command
  merge       create a new manifest by updating a manifest with the contents of other manifests
  rewrite     rewrite a manifest, updating the prefix of all paths

Flags:
  -h, --help   help for manifest

Use "manifest [command] --help" for more information about a command.
```

## Warts with relative paths

Normally, all paths are considered relative to the working directory (i.e. the directory you're in when you call the `manifest` binary).
However, all paths in test manifests loaded through `kite-go/lang/python/pythonresource/testing.go` are considered relative to the manifest file.
This means you should generally ensure two things:

1. Always use the same working directory when generating/manipulating manifests containing relative paths.
2. When generating test manifests, `cd` to a new directory that will hold your test manifest as well as all resource data (relative to that manifest).
   Then just sync that directory directly into `kiteco`.

## Building a new resource

As mentioned previously, you should usually stay in the same working directory for this process.

1. Build a new manifest file & the relevant resource data using the corresponding binary from `kite-go/lang/python/cmds/build/...`
    ```
    > ./argspecs manifest.json path/to/data/dir
    ```
2. After verifying the newly generated data, rewrite the manifest to point at the default resource manager S3 bucket.
    ```
    > ./manifest rewrite manifest.json path/to/data/dir -o manifest_remote.json
    ```
3. Verify the new `manifest_remote.json`, and then sync the resource data to the new location.
    ```
    > ./manifest copydata manifest.json manifest_remote.json
    ```
4. Merge the new paths into the committed manifest.
    ```
    > ./manifest merge committed_manifest.json manifest_remote.json
    ```
