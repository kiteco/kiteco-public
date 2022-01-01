/* global gapi */

import scriptLoader from 'react-async-script-loader'

/* ==={ CONSTANTS AND KEYS }=== */

const CLIENT_IDS = {
  live: 'XXXXXXX.apps.googleusercontent.com',
  test: 'XXXXXXX.apps.googleusercontent.com'
}

const getClientID = () => {
  if (process.env.REACT_APP_ENV === 'development') {
    return CLIENT_IDS.test
  }
  if (process.env.NODE_ENV === 'production') {
    return CLIENT_IDS.live
  }
  return CLIENT_IDS.test
}

const CLIENT_ID = getClientID()

const SCOPES = "https://www.googleapis.com/auth/contacts.readonly"

/* ==={ API SCRIPT }=== */

/**
 * This wraps a container or a component
 * to load the Google js API
 * Multiple containers can use this wrapper
 * without loading the script twice.
 *
 * Examples:
 *   wrapGoogleLoad(Component)
 *   wrapGoogleLoad(connect(mapStateToProps)(Container))
 */
export const wrapGoogleLoad = scriptLoader("https://apis.google.com/js/api.js")

/* ==={ INITIALIZATION }=== */

const clientInit = () =>
  gapi.client.init({
    clientId: CLIENT_ID,
    scope: SCOPES,
  })

const clientLoad = () => {
  gapi.load('client:auth2', clientInit)
}

/**
 * Init the gapi  object with our api key.
 *
 * This must be found in both componentDidMount and
 * componentDidUpdate
 *
 * Ensure that a component or container that users this
 * method is wrapped with `wrapGoogleLoad` above.
 *
 * Example:
 *
 *    componentDidMount() {
 *      initGoogle({ props: this.props })
 *    }
 *
 *    componentDidUpdate(props) {
 *      initGoogle({ props, prevProps: this.props })
 *    }
 *
 */
export const initGoogle = ({ props, prevProps = {} }) => {
  const { isScriptLoaded: load, isScriptLoadSucceed: succeed } = props
  const { isScriptLoaded: loaded, isScriptLoadSucceed: succeeded } = prevProps
  if ((load && succeed) && !(loaded && succeeded)) {
    clientLoad()
  }
}

/* ==={ AUTHENTICATION STATE }=== */

export const signIn = () =>
  gapi.auth2.getAuthInstance().signIn()
  .then(user => {
    return user.getAuthResponse(true)
  })

/* ==={ API CALLS }=== */


/**
 * Recommended usage:
 * signIn()
 * .then(getContacts)
 * .then(({ success, data }) => { data })
 */
export const getContacts = authResponse =>
  gapi.client.request({
    path: "/m8/feeds/contacts/default/full",
    params: {
      "alt": "json",
      "max-results": 2000,
    }
  }).then(
    response => {
      // TODO: this api returns messy deeply nested
      // data. For now, just try and catch null pointers.
      // In the future, consider using:
      // https://github.com/paularmstrong/normalizr
      try {
        return {
          success: true,
          data: response.result.feed.entry
          .map(e => ({
            aliases: e["gd$email"],
            title: e.title["$t"],
          }))
            .reduce((emails, bundle) => [
              ...emails,
              ...bundle.aliases.map(g => ({
                title: bundle.title,
                email: g.address,
              }))
            ], []),
        }
      } catch (error) {
        return {
          success: false,
          error,
        }
      }
    },
    error  => ({
      success: false,
      error,
    }),
  )

