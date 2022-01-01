/*
  Wrapper for Customer.io analytics client.
*/
import CIO from "../customerio-node"

var currentId = ""
var cio = null

const SITE_IDS = {
  prod: "XXXXXXX",
  dev: "XXXXXXX",
}

const API_KEYS = {
  prod: "XXXXXXX",
  dev: "XXXXXXX",
}

const getCredentials = () => {
  if (process.env.REACT_APP_ENV === "development") {
    return [SITE_IDS.dev, API_KEYS.dev]
  }
  if (process.env.NODE_ENV === "production") {
    return [SITE_IDS.prod, API_KEYS.prod]
  }
  return [SITE_IDS.dev, API_KEYS.dev]
}


export const load = () => {
  if (!cio) {
    cio = new CIO(...getCredentials())
  }
}

export const identify = id => {
  currentId = id
  cio && cio.identify(id)
}

export const getId = () => {
  return currentId
}

export const track = ({ event, props }) => {
  const metadata = { name: event, data: props }
  if (currentId) {
    cio && cio.track(currentId, metadata).catch(() => {})
  } else {
    cio && cio.trackAnonymous(metadata).catch(() => {})
  }
}

export const reset = () => {
  currentId = ""
}
