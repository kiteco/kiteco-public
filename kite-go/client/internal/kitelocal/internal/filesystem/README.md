# Filesystem Manager
The `filesystem` has two managers, the `Manager` and the `LibraryManager`.
The `Manager` consists of three components:
- `libraryWalker`
- `watcher`
- `LocalFS`

## Library Walker
On init, the library walk is started. It runs in the background as a goroutine and is only run once. It performs a walk of the filesystem rooted at the user's home directory. It does not walk [directories that are filtered](https://github.com/kiteco/kiteco/blob/master/kite-golib/filters/filter.go#L36). If it encounters a [library directory](https://github.com/kiteco/kiteco/blob/master/kite-golib/filters/filter.go#L62), it stores it and doesn't descend further into that directory. These libraries are then requested on each index build.

## Library Manager
During an index build, the builder attempts to discover as many library locations as it can to use when handling imports in file selection. These libraries are discovered using the `LibraryManager`. The library manager has many known [library types](https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/internal/filesystem/libraries.go#L21) that it finds using a variety of methods. Virtual env locations are found by pattern-matching; it reads `sys.Path` to find installation-dependent default python paths; and it uses the directories found in the library walk. Additionally, the user can specify library locations that Kite may not find on its own in `<kiteDir>/libraries`. Apart from the directories discovered by the library walk, the other methods of library discovery are run on each index build to get the most recent information available.

## Watcher
The watcher watches the user's home directory for filesystem changes to python files and sends requests for index builds to the indexer via an event channel. Events are not sent for files that are not accepted (e.g. non-python files) or for changes in [filtered directories](https://github.com/kiteco/kiteco/blob/master/kite-golib/filters/filter.go#L36).

## LocalFS
The LocalFS implements an API the index builder uses to access the user's local filesystem. It exposes functions `Stat`, `Glob`, and `Walk`. 

During file selection, `Walk` finds children of the parent directory of the file being indexed (until the project file limit is hit). If it encouters a [filtered directory](https://github.com/kiteco/kiteco/blob/master/kite-golib/filters/filter.go#L36), it does not recursively walk its entries (it does apply the `WalkFn` to the non-directory entries, however). If it encounters a library directory, it skips it entirely (it does not apply the `WalkFn` to any entries).

`Glob` is used for pattern-matching by the `LibraryManager` to find virtual envs. It is also used by file selection when attempting to handle imports. If a `Stat` for a package fails, it uses `Glob` to check if the package is an `.egg`.

`Stat` is used during file selection to find project paths (ancestor directories that do not contain `__init__.py`). It is also used when attempting to resolve imports. In this case, `Stat` can be called on directories in path imports, on project paths, and on library directories. `Stat` does not actually call `os.Stat` for files in [filtered directories](https://github.com/kiteco/kiteco/blob/master/kite-golib/filters/filter.go#L36).
