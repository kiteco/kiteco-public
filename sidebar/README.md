This project was bootstrapped with [Create React App](https://github.com/facebookincubator/create-react-app).

#Commands:

```bash
$ npm install
```
To install. Should be setup to run `npm run dev` successfully.

However, if it errors out, you may want to try installing the following packages globally:
```bash
$ npm install -g concurrently electron electron-builder react-scripts cross-env wait-on
```

```bash
# you may need to run `npm run build` at least once before running this command. It requires `build/electron.js` which obviously won't be there without at least one build step
$ npm run dev
```

Starts the dev server and launches a dev electron

Note: to run the dev server + dev electron concurrently with a dev kited (e.g. to test how kited endpoints parse data that the electron app submits):
  - kill whatever kited engine you're currently running
  - build via XCode
  - exit the sidebar when it displays (keeping the engine that was booted up)
  - `npm run dev` -> the dev server will now use the development kited engine

```bash
$ npm run pack
```

Creates a production version of the app

If you need to test something in a built electron app, versus something that can be tested via `npm run dev`, if the `process.env.ELECTRON_ENV === "development"` condition is met, then the built app will open with a development console, which you can log to
