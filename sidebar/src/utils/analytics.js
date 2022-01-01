import * as cio from "./customer-io"
import * as mixpanel from "./mixpanel"

const SOURCE = "sidebar"

const getCurrentTime = () => {
  const d = new Date()
  return Math.floor(d.getTime() / 1000)
}

// canUse mirror metricsEnabled
let canUse = false
export const setCanUse = (val) => {
  canUse = val
}

export const track = ({ event, props }) => {
  if (process.env.REACT_APP_ENV === 'development') {
    console.log(`tracking ${event}`, props)
  }

  if (!canUse) {
    return
  }

  const sent_at = getCurrentTime()
  cio.track({
    event,
    props: {
      source: SOURCE,
      sent_at: sent_at,
      user_id: cio.getId(),
      ...props,
    },
  })
  mixpanel.track({
    event,
    props: {
      source: SOURCE,
      sent_at: sent_at,
      user_id: cio.getId(),
      ...props,
    },
  })
}

let hasLoaded = false
export const load = () => {
  if (process.env.REACT_APP_ENV === 'development') {
    console.log('load', hasLoaded, canUse)
  }
  if (!hasLoaded && canUse) {
    mixpanel.load()
    cio.load()
    hasLoaded = true
  }
  return hasLoaded
}

export const register_once = props => {
  canUse && mixpanel.register_once(props)
}

export const get_distinct_id = () => {
  return canUse && mixpanel.get_distinct_id()
}

export const identify = id => {
  if (!canUse) {
    return
  }
  cio.identify(id)
  mixpanel.identify(id)
}

export const reset = () => {
  if (!canUse) {
    return
  }
  cio.reset()
  mixpanel.reset()
}
