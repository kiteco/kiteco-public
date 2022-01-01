import fetch from 'isomorphic-fetch'

import { localhostProxy, kitedStatusPath } from '../utils/urls'

const apiFetchOptions = {
  credentials: 'same-origin',
}

const noCacheHeaders = {
  "cache-control": "no-store",
  "pragma": "no-cache",
}

const defaultHeaders = {
  "accept": "text/plain,application/json",
}


/* ==={ HELPERS }=== */

/**
 * Helper method to resolve the body of a response
 * Intended usage:
 *
 *   fetch(url)
 *   .then(resolveBody)
 *   .then( ({ response, json, text }) => {
 *      // Do something with response, json, text
 *   })
 *
 * Note: either `json` or `text` will be returned
 * but not both.
 */
export const resolveBody = response => {
  // We lowercase to handle responses with charset=utf-8 and charset=UTF-8
  // similarly.
  const contentType = response.headers.get('Content-Type')
    ? response.headers.get('Content-Type').toLowerCase()
    : response.headers.get('Content-Type')

  switch (contentType) {
    case 'application/json':
    case 'application/json; charset=utf-8':
      return response.json()
        .then(json => ({
          response,
          json,
        }))
    case 'text/plain':
    case 'text/plain; charset=utf-8':
      return response.text()
        .then(text => ({
          response,
          text,
        }))
    default:
      console.error(`fetch request returned response with content-type: ${contentType}`)
      return response.text()
        .then(text => ({
          response,
          text: null,
          incorrectTypeContent: text,
        }))
  }
}

//the proxy error text from
//https://github.com/facebook/create-react-app/blob/next/packages/react-dev-utils/WebpackDevServerUtils.js
const needsRethrow = (body) => {
  return body.text === 'server error' &&
    body.incorrectTypeContent.indexOf('Proxy error: Could not proxy request') !== -1
}

//TODO: still need a better definitional version of health for kited wrt how it interacts with clients
// of it
export const ignoreUrlForResponseCode = (resp) => {
  return resp.response.status < 500 || (resp.response.url && !resp.response.url.includes(kitedStatusPath()))
}

//Would like to eventually refactor to, with discrimination, handle a variety of
//codes. Right now, app expects 401's from /api/account/user on logged-out condition
//to be handled as if nothing is occurring, so simple 'response.response.ok' check is
//insufficient
export const isOK = (response) => {
  return response.response.ok ||
    response.response.status < 500 //e.g. Service Unavailable -> what's given from /clientapi/health
}

/**
 * Ensure that we send session credentials with all
 * requests if we have them. This also wraps the fetch
 * interface such that we pass parameters around in objects
 * instead of with parameters.
 */
export const authFetch = ({ url, options={}}) =>
  fetch(
    // for dev, proxy through dev server @ localhost:3000 to avoid CORS
    // for production, hit localhost directly
    (url.startsWith("/") && process.env.NODE_ENV === "production")
      ? localhostProxy(url)
      : url,
    {
      ...apiFetchOptions,
      ...options,
    },
  )
    .then(resolveBody)
    .then((body) => {
    // As in dev mode we're going through a proxy, we don't get a rejected
    // promise when kite is unreachable, instead we get a 500 error with
    // a some kind of network error message in it. By rethrowing that at this point we
    // stay consistent with production env.
      if (process.env.NODE_ENV !== "production") {
        if (needsRethrow(body)) {
          throw body
        }
      }
      return body
    })

/* ==={ DATA PREP }=== */

export const createQueryURL = (url, queries) => {
  if (queries) {
    const qs = Object.keys(queries)
      .map(k => `${encodeURIComponent(k)}=${encodeURIComponent(queries[k])}`)
      .join('&')
    return `${url}?${qs}`
  }
  return url
}

/*
 * createJson creates stringifies JSON objects
 * and prepares them for use inside of `fetch`'s
 * options parameter
 */
export const createJson = (obj, options={}) => ({
  headers: {
    'Content-Type': 'application/json',
    ...options.headers,
  },
  body: JSON.stringify(obj),
  ...options,
})

/*
 * createFormData takes in a object and turns it into
 * formData ready to be used inside of `fetch`'s object parameter
 */
export const createFormData = (obj, options={}) => {
  const data = new FormData()
  for (let key in obj) {
    if (obj.hasOwnProperty(key)) {
      data.append(key, obj[key])
    }
  }
  return {
    body: data,
    ...options,
  }
}

/* ==={ METHODS }=== */

export const GET = ({ url, options={}, queries=false }) =>
  authFetch({
    url: createQueryURL(url, queries),
    options: {
      method: "GET",
      ...options,
      headers: {
        ...noCacheHeaders, // when doing XHR get requests, unlikely to want to cache
        ...defaultHeaders,
        ...options.headers,
      },
    },
  })

export const POST = ({ url, options={}}) => {
  return authFetch({
    url,
    options: {
      method: "POST",
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    },
  })
}

export const PUT = ({ url, options={}}) =>
  authFetch({
    url,
    options: {
      method: "PUT",
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    },
  })

export const DELETE = ({ url, options={}}) =>
  authFetch({
    url,
    options: {
      method: "DELETE",
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    },
  })

export const PATCH = ({ url, options={}}) =>
  authFetch({
    url,
    options: {
      method: "PATCH",
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    },
  })

/* ==={ HELPERS }=== */

export const errTimedOut = "errTimedOut"
export const timeoutAfter = (dispatchRequest, ms) => {
  return Promise.race([
    dispatchRequest(),
    new Promise((resolve) => {
      setTimeout(resolve, ms, errTimedOut)
    }),
  ])
}
