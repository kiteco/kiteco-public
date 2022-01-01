# Development Guide

## React App
The base Kite Answers code lives in `./web/render`. A complementary React app for previewing lives in `./web/preview`.

To test any changes, you'll need to:
1. run `npm link` in the `./web/render` directory, then `npm link @kiteco/kite-answers-renderer`, in either the Preview app, or the Webapp.
   1. This only needs to be done once, **unless the dependency is re-installed** (during the deployment process, we run `npm i @kiteco/kite-answers-renderer@latest` to grab the latest changes).
2. Build the app, either with `npm run build-dev`, or `npm run build-prod` for an obfuscated output file. **This needs to happen each time the renderer is updated.**
3. Run the preview app or webapp with `npm run`. 
   1. For the Webapp, optional args may be passed in to hit different hostnames, refer to each package's `package.json` for details.
   2. The preview app's endpoints are defined in `./web/preview/utils/constants.js`. Until the process is improved, we'll need to update the values manually if another hostname is desired.

## Preview Server
To test changes with the go preview server:
1. Execute `go run ./go/cmds/preview-server/`
2. You can open `localhost` in your browser to render the currently bindata'd React bundle.

To deploy changes to the preview server:
1. Make sure you have credentials to our team's prviate npm repo
2. If any changes were made in the `render` package, from the project's root dir, run `npm version {x}` where `x` increments the current version in `package.json`, then `npm publish`.
3. Go to the root `kite-answers` directory. Run `make preview-app && make assets && make deploy-preview`.

## Publishing Flow

To run the publishing command line tool, you'll need:
1. AWS CLI
2. AWS Credentials
3. Docker Desktop (for first time setup, go to `./go/execution/sandbox/` and run the Makefile to setup dependencies).
4. Access to https://github.com/kite-answers/answers

For now, we'll need to check out and pull the repo locally to grab the latest content.

To run the actual tool:
1. Run `go run ./go/publish/main.go {...}` where `{...}` is the path to the Kite Answers posts path.
2. Wait for the output file to be generated. (Current implementation will take awhile).
3. Upload file to S3 in the bucket `kite-data/kite-answers`.
