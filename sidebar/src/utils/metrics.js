import fetch from 'isomorphic-fetch'
import { localhostProxy, countersPath } from '../utils/urls'

export const metrics = {
  incrementCounter: (counterName) => {
    const COUNTERS_ENDPOINT_URL = countersPath()
    fetch(
      (process.env.NODE_ENV === "production")
      ? localhostProxy(COUNTERS_ENDPOINT_URL)
      : COUNTERS_ENDPOINT_URL
      , {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: counterName,
          value: 1,
        }),
      }
    )
  },
}
