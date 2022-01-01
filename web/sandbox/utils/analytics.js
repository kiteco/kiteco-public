/* analytics */

/*
  We assume a global analytics object in the environment presumably provided by a Segment script 
*/

const getCurrentTime = () =>  {
  const d = new Date()
  return Math.floor(d.getTime()/1000)
}

const wrapReady = func => props => analytics && analytics.ready && analytics.ready(() => func(props))

const trackImpl = ({ event, props }) => {
  analytics && analytics.track && analytics.track(event, {
    source: SOURCE,
    sent_at: getCurrentTime(),
    user_id: analytics.user().id(),
    ...props,
  })
}

export const track = wrapReady(trackImpl)

export const register_once = props => {
  analytics && analytics.identify && analytics.identify(props)
}

export const identify = id => {
  analytics && analytics.identify && analytics.identify(id)
  // need to make sure fbq has been loaded from GTM
  const facebookIdentify = () => {
    if (window.fbq) {
      window.fbq('init', 'XXXXXXX', { uid: id })
    } else {
      setTimeout(facebookIdentify, 100)
    }
  }
}

export const reset = () => {
  analytics && analytics.reset && analytics.reset()
}

export const alias = id => {
  analytics && analytics.alias && analytics.alias(id)
}

