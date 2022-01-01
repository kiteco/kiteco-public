import ReactGA from 'react-ga'
import { LOCATION_CHANGE } from 'connected-react-router'
ReactGA.initialize('UA-53431456-1', {
  gaAddress: "/static/_kite-google.js",
})
ReactGA.plugin.require('GTM-MGZFQ4H')

export const logGA = location => {
  ReactGA.pageview(location.pathname)
}

// submits route changes to google analytics
// This NEEDS to be placed before the routerMiddleware!
export const gaMiddleware = store => next => action => {
  if (action.type === LOCATION_CHANGE) {
    logGA(action.payload)
    // https://support.google.com/optimize/answer/7008840?hl=en
    setTimeout(
      () => window.dataLayer.push({
        event: "optimize.activate",
      }), 0
    )
  }
  return next(action)
}

export const event = props => ReactGA.event(props)
