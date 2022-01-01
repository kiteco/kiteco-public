import React, {Suspense} from "react";
import { render } from "react-dom";
import { createStore, applyMiddleware, compose } from "redux";
import thunkMiddleware from "redux-thunk";
import { Provider } from "react-redux";
import { composeWithDevTools } from "redux-devtools-extension";
import persistState from "redux-localstorage";

// routing
import { createBrowserHistory as createHistory } from "history";
import { Route, Redirect, Switch } from "react-router-dom";
import {
  ConnectedRouter as Router,
  routerMiddleware
} from "connected-react-router";

//ErrorBoundary
import ErrorBoundary from "./components/ErrorBoundary";

// loaded components
import Head from "./components/Head";
import Notifications from "./components/Notifications";

// lazy loaded components
import Login from "./pages/Login";
import Settings from "./pages/Settings";
import Invite from "./pages/Invite";
import ResetPassword from "./pages/ResetPassword";
import VerifyEmail from "./pages/VerifyEmail";
import NotFound from "./pages/NotFound";

import AnonymousID from "./components/AnonymousID";
import UTMTracking from "./components/UTMTracking";
import HistoryListener from "./components/HistoryListener";

import reducer from "./redux/reducers";

import { gaMiddleware } from "./utils/ga";
import { analyticsMiddleware } from "./utils/analytics";

import {
  oldToNewPath,
  oldToNewExamplesPath,
  hasRawId
} from "./utils/route-parsing";

import { DEVELOPMENT } from "./utils/development";
import Rollbar from "./utils/rollbar";

import "./assets/styles/global.css";
import "./assets/styles/less/global.less";

import { LOAD_CACHED_COMPLETIONS } from "./redux/reducers/cached-completions";
// we assume it's here, because we can't import conditionally
import CACHED_COMPLETIONS from "./assets/data/precaching/completions.json";

import TagManager from 'react-gtm-module'

// - polyfills

import "babel-polyfill";
if (typeof Promise === 'undefined') {
  // Rejection tracking prevents a common issue where React gets into an
  // inconsistent state due to an error, but it gets swallowed by a Promise,
  // and the user has no idea what causes React's erratic future behavior.
  require('promise/lib/rejection-tracking').enable();
  window.Promise = require('promise/lib/es6-extensions.js');
}

// fetch() polyfill for making API calls.
require('whatwg-fetch');

// AbortController polyfill for cancelling API calls
require('abortcontroller-polyfill/dist/polyfill-patch-fetch')

// Object.assign() is commonly used with React.
// It will use the native implementation if it's present and isn't buggy.
Object.assign = require('object-assign');

// In tests, polyfill requestAnimationFrame since jsdom doesn't provide it yet.
// We don't polyfill it in the browser--this is user's responsibility.
if (process.env.NODE_ENV === 'test') {
  require('raf').polyfill(global);
}

const AnswersContainer = React.lazy(() => import("./pages/Answers"));
const Docs = React.lazy(() => import("./pages/newDocs/components/Docs"));

// Google Tag Manager
TagManager.initialize({
  gtmId: 'GTM-NMNRG7'
});

// only include redux devtools chrome extension with dev builds
const envCompose = DEVELOPMENT ? composeWithDevTools : compose;

// create history object
export const history = createHistory();

//need to register the listener here - otherwise, the interception and adding
//the action value to the state occurs after being handled by the middleware
//purpose is to get to breadcrumbs in HowToExamples working sanely in the case
//a user uses the Back/Forward browser navigations
const historyUnlisten = history.listen((location, action) => {
  if (!location.state) {
    location.state = {};
  }
  location.state.navAction = action;
});

const routerReduxMiddleware = routerMiddleware(history);

// middleware
const middleware = applyMiddleware(
  thunkMiddleware,
  gaMiddleware,
  analyticsMiddleware,
  routerReduxMiddleware
);

// local storage
const storage = persistState(
  ["promotions", "comments", "history", "starred", "stylePopup"],
  { key: "kite" }
);

const store = createStore(reducer(history), envCompose(middleware, storage));

//initialize store
store.dispatch({
  type: LOAD_CACHED_COMPLETIONS,
  completions: CACHED_COMPLETIONS
});

//error handler for errors (e.g. errors thrown in handlers) that won't
//get caught by componentDidCatch
window.addEventListener("error", function(event) {
  const payload = {
    ...event,
    stack: event.error?.stack
  };
  Rollbar.handleException(payload);
  //store handling
  store.dispatch({
    type: "APP_EXCEPTION"
  });
});

const VALID_LANGUAGES = "python|js";

const PublicRoutes = (
  <Switch>
    { DEVELOPMENT &&
      <Route
        exact path="/"
        render={props => "The home page is now served by WordPress."} />
    }

    <Redirect from="/ref/*" to="/" />
    <Route
      path="/docs"
      render={({ location }) => {
        //parse id from location.pathname
        return <Redirect to={oldToNewPath(location.pathname)} />;
      }}
    />
    <Route
      path="/examples"
      render={({ location }) => {
        return <Redirect to={oldToNewExamplesPath(location.pathname)} />;
      }}
    />


    <Route path={`/python/answers/:slug`} component={AnswersContainer} />
    <Route
      path={`/:language(${VALID_LANGUAGES})`}
      render={props => {
        if (hasRawId(props.location.pathname)) {
          return <Redirect to={oldToNewPath(props.location.pathname)} />;
        }
        return <Docs {...props} />;
      }}
    />
    <Route path="/checkout" component={React.lazy(() => import("./pages/Checkout"))} />
    <Route path="/settings" component={Settings} />
    <Route path="/login" component={Login} />
    <Route path="/invite" component={Invite} />
    <Route path="/reset-password" component={ResetPassword} />
    <Route path="/verify-email" component={VerifyEmail} />
    <Route component={NotFound} />
  </Switch>
);

render(
  <Provider store={store}>
    <ErrorBoundary>
      <Router history={history}>
        <Suspense fallback={<div>Loading...</div>}>
          <div className="main">
            <HistoryListener unlisten={historyUnlisten} />
            <AnonymousID />
            <UTMTracking />
            <Head />
            {PublicRoutes}
            <Notifications />
          </div>
        </Suspense>
      </Router>
    </ErrorBoundary>
  </Provider>,
  document.getElementById("root")
);
