export const LOG_RESPONSE_TIME = 'log_response_time'
const logTime = (obj, key) => {
  obj[key] = Date.now()
}

export const performPreMiddleware = (middleware) => {
  for (const key in middleware) {
    if(!middleware[key]) {
      middleware[key] = {}
    }
    switch(key) {
      case LOG_RESPONSE_TIME:
        logTime(middleware[key], "start")
        break
      default:
        break
    }
  }
  return middleware
}

export const performPostMiddleware = (middleware) => {
  for(const key in middleware) {
    if(!middleware[key]) {
      middleware[key] = {}
    }
    switch(key) {
      case LOG_RESPONSE_TIME:
        logTime(middleware[key], "end")
        break
      default:
        break
    }
  }
  return middleware
}