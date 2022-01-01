/* eslint-disable no-throw-literal */
import fetch from 'isomorphic-fetch'

import { performPostMiddleware, performPreMiddleware } from './fetch-middleware'

const AbortController = window.AbortController

const apiFetchOptions = {
  credentials: 'include',
};

const noCacheHeaders = {
  "cache-control": "no-store",
  "pragma": "no-cache",
};

/**
 * This is used to prefix all root level urls with given
 * prefix if specified at build time.
 * Example usage:
 *
 * $ REACT_APP_BACKEND="https://alpha.kite.com" npm build
 */
const prefixURL = (url, suppliedPrefix) => {
  const prefix = suppliedPrefix || process.env.REACT_APP_BACKEND
  if (prefix && url.substring(0, 1) === "/") {
    url = prefix + url;
  }
  return url
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
export const resolveBody = (middleware) => response => {
  if (response.headers.has('Content-Type') && response.headers.get('Content-Type').includes('application/json')) {
    return response.json()
      .then(json => {
        return {
          response,
          json,
          middleware,
        }
      })
  } else {
    return response.text()
      .then(text => {
        return {
          response,
          text,
          middleware,
        }
      })
  }
}

const abortWithTimeout = timeout => {
  const controller = new AbortController()
  setTimeout(() => {
    controller.abort()
  }, timeout)
  return controller.signal
}

/**
 * Ensure that we send session credentials with all
 * requests if we have them. This also wraps the fetch
 * interface such that we pass parameters around in objects
 * instead of with parameters.
 */
export const authFetch = ({ url, urlPrefix, options={}, middleware }) => {
  if(options.hasOwnProperty('timeout') && options.timeout) {
    const timeout = options.timeout
    delete options.timeout
    options.signal = abortWithTimeout(timeout)
  }
  if(middleware) {
    middleware = performPreMiddleware(middleware)
  }
  return fetch(prefixURL(url, urlPrefix),
    {
      ...apiFetchOptions,
      ...options,
    },
  )
  .catch(error => {
    throw { error, middleware: performPostMiddleware(middleware) }
  }) //will want to include middleware
  .then(resolveBody(middleware))
  .then(res => {
    res.middleware = performPostMiddleware(res.middleware)
    return res
  })
}


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
  const data = new FormData();
  for (let key in obj) {
    if (obj.hasOwnProperty(key)) {
      data.append(key, obj[key]);
    }
  }
  return {
    body: data,
    ...options,
  }
}

/* ==={ METHODS }=== */

export const GET = ({ url, options={}, queries, urlPrefix, timeout, middleware }) =>
  authFetch({
    url: createQueryURL(url, queries),
    urlPrefix,
    options: {
      headers: {
        ...noCacheHeaders, // when doing XHR get requests, unlikely to want to cache
        ...options.headers,
      },
      method: "GET",
      ...options,
      timeout,
    },
    middleware,
  })

export const POST = ({ url, options={}, urlPrefix, timeout, middleware }) =>
  authFetch({
    url,
    urlPrefix,
    options: {
      method: "POST",
      ...options,
      timeout,
    },
    middleware,
  })

export const PUT = ({ url, options={}, urlPrefix, timeout, middleware }) =>
  authFetch({
    url,
    urlPrefix,
    options: {
      method: "PUT",
      ...options,
      timeout,
    },
    middleware,
  })

export const DELETE = ({ url, options={}, urlPrefix, timeout, middleware }) =>
  authFetch({
    url,
    urlPrefix,
    options: {
      method: "DELETE",
      ...options,
      timeout,
    },
    middleware
  })

export const PATCH = ({ url, options={}, urlPrefix, timeout, middleware }) =>
  authFetch({
    url,
    urlPrefix,
    options: {
      method: "PATCH",
      ...options,
      timeout,
    },
    middleware,
  })
