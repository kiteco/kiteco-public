var request = require('request')

const metrics = {

  trackSidebarFocused() {
    const property = 'sidebar_focused'

    const headers = {
      'Content-Type': 'application/json',
    }

    const options = {
      url: "http://localhost:46624/clientapi/metrics/counters",
      method: 'POST',
      headers,
      form: JSON.stringify({
        "name" : property,
        "value" : 1,
      })
    }

    request(options, (error, response, body) => {
    })

  }

}

module.exports = {
  metrics,
}
