/*
  Wrapper for Mixpanel analytics client.
*/
import mixpanel from "mixpanel-browser"

const TOKENS = {
  live: "XXXXXXX",
  test: "XXXXXXX",
}

const getMixpanelToken = () => {
  if (process.env.REACT_APP_ENV === "development") {
    return TOKENS.test
  }
  if (process.env.NODE_ENV === "production") {
    return TOKENS.live
  }
  return TOKENS.test
}

const TOKEN = getMixpanelToken()

let hasLoaded = false
export const load = () => {
  if (!hasLoaded) {
    mixpanel.init(TOKEN, { "ignore_dnt": true })
    hasLoaded = true
  }
}

export const get_distinct_id = () => {
  return mixpanel.get_distinct_id()
}

export const track = ({ event, props }) => {
  hasLoaded && mixpanel.track(event, props)
}

export const register_once = props => {
  hasLoaded && mixpanel.register_once(props)
}

export const identify = id => {
  hasLoaded && mixpanel.identify(id)
}

export const reset = () => {
  hasLoaded && mixpanel.reset()
}

export const alias = id => {
  hasLoaded && mixpanel.alias(id)
}

export const people_set = props => {
  hasLoaded && mixpanel.people.set(props)
}
