# Kite Web App

## Getting started

### 1. Install Npm

- MacOS : `brew install node`
- Ubuntu: The version available in base repo is too old so it's easier to use nodejs repo:
```bash
#Download a script that adds the right repo in sources lists
curl -sL https://deb.nodesource.com/setup_12.x | sudo -E bash - 
sudo apt install nodejs
```

### 2. Install packages

You first need to login to the private npm repo:
`npm login` and use credentials available in the Credentials quip doc.

In the project root directory, run:

```bash
npm install
```

If you get the error 
```
npm ERR! typeerror Error: Missing required argument #1
npm ERR! typeerror     at andLogAndFinish (/usr/share/npm/lib/fetch-package-metadata.js:31:3)
```
Try upgrading your npm version with `npm install -g npm@latest` (don't use sudo, look on google for `EACCESS error` if you get error upgrading it)

### 3. Start the dev server

In the project root directory, run:

```bash
npm start
```

This uses `staging.kite.com` as the backend.

### Deployment

To deploy this website, please see `kiteco/scripts/deploy_webapp.sh` or consider the slackbuildbot `solness`.

## Recommended tooling

It is highly recommended that you install and check out [Redux Devtools Extension](http://extension.remotedev.io/#installation) and [React Developer Tools](https://github.com/facebook/react-devtools).

## Recommended development practices

This guide will reference a repo that contains a small app that implements many of the practices below: [Patient-List](https://github.com/intrepidlemon/patient-list). Keep in mind that these tips below are flexible and if you find a style or technique that suits you better and creates more understandable, concise code, please update the recommendations below and share.

### Javascript

- Use ES6 syntax for brevity and clarity

### Components

- Strive to use [functional components](https://hackernoon.com/react-stateless-functional-components-nine-wins-you-might-have-overlooked-997b0d933dbc) when possible.
    - As a guideline, class-based components should only be used when a component needs to store some type of transient state; examples include a form component, which needs to store user input, or a carosel component which has to store which image it is currently on.
- When a component needs access to the global store, or to dispatching actions, use redux's [`connect`](https://www.sohamkamani.com/blog/2017/03/31/react-redux-connect-explained/).

- Every component should strive to do **one** thing well. For example, look at [`RangePlot`](https://github.com/intrepidlemon/patient-list/blob/master/src/components/RangePlot/index.js)
    - Notice how `RangePlot` implements a single component that plots a point on a range.
    - Notice how the implementation of `RangePlot` is split up into two components:
        - `RangePlot`, the parent component which controls the component, checks for missing data and presents the appropriate view given this information. One might think of this component as being a "controller" in the MVC model
        - `Plot` which is a view component, containing no data manipulation or processing. One might consider this the "view" in the MVC model

### CSS

All new CSS should be use CSS Modules (instead of BEM).
