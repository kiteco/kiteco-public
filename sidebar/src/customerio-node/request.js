const request = require('request')
const TIMEOUT = 10000

class Request {
  constructor(siteid, apikey) {
    this.siteid = siteid
    this.apikey = apikey
    this.auth = `Basic ${new Buffer(
      this.siteid + ':' + this.apikey,
      'utf8'
    ).toString('base64')}`
    this._request = request
  }

  options(uri, method, data) {
    const headers = {
      Authorization: this.auth,
      'Content-Type': 'application/json'
    }
    const body = data ? JSON.stringify(data) : null
    const options = { method, uri, headers, body, timeout: TIMEOUT }

    if (!body) delete options.body

    return options
  }

  handler(options) {
    return new Promise((resolve, reject) => {
      this._request(options, (error, response, body) => {
        if (error) return reject(error)

        let json = null
        try {
          if (body) json = JSON.parse(body)
        } catch (e) {
          const message = `Unable to parse JSON. Error: ${e} \nBody:\n ${body}`
          return reject(new Error(message))
        }

        if (response.statusCode == 200 || response.statusCode == 201) {
          resolve(json)
        } else {
          reject({
            message: (json.meta && json.meta.error) || 'Unknown error',
            statusCode: response.statusCode,
            response: response,
            body: body
          })
        }
      })
    })
  }

  put(uri, data = {}) {
    return this.handler(this.options(uri, 'PUT', data))
  }

  destroy(uri) {
    return this.handler(this.options(uri, 'DELETE'))
  }

  post(uri, data = {}) {
    return this.handler(this.options(uri, 'POST', data))
  }
}

export default Request
