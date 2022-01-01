## Description ##
Tools and pipelines for testing python type inference coverage based on code from our live users.
ONLY RUN ON AWS INSTANCES

## Tools ##
- `make download` -- given a list of `userID`s, downloads all of the python files for the users (across all their machines) and 
  writes them locally to `.artifacts/corpus`, e.g: `./env.sh make download`
- `make count` -- given an input corpus (`.artifacts/corpus`), count the number of (global) modules, attributes, etc, 
  that were not resolved and write out the results, e.g: `make count`
- `submodules show` -- given a set of counts from `count` display them in a web browser, e.g: from `/submodules` run `submodules show ../.artifacts/counts.json.gz`
- `make coverage` -- given a corpus measure the coverage of static analysis on attribute expressions
- `coverage show` -- given a corpus calculate the coverage of static analysis on attribute expressions and show the results in a web browser

## Users ##
- Current set of users are assorted users from `us-west-1` and had `kite-sig-vis` in october

## Notes ##
- ONLY RUN ON AWS INSTANCES
- Get user IDs from http://invites.kite.com/metrics 
    - global metrics -> click on any square gives list of users, mix panel url also happens to be their user ID,
- Get relvant environment variables from Juan (`env.sh` script)
- If running on a test instance then need to update aws credentials with credentials from local machine

## TODO ##
-  Deal with same user in multiple regions
-  Deal with multiple user machines (right now we just grab all files for all user machines)
-  Missing attributes from global nodes that are used as bases of a user derived class
-  What about methods/fields defined in one class (that reference a global node) that are then used in a different module?
-  Indirect imports (e.g user module imports global package, then references global package via local module, in an import or as an attr)
-  We have an attribute attached to a global node as a member but the node for the member itself is nil
-  Deal with memory issues for `coverage show` tool
-  Add flag to `coverage show` tool to optionally also run single file resolver to get better estimate of how we do on `user-node`