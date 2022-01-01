window.doorbellOptions = {
  id: 'XXXXXXX',
  appKey: 'XXXXXXX',
  hideEmail: false,
  hideButton: true,
  tags: 'electron',
};

export const load = () => {
  // script directly from doorbell
  (function (w, d, t) {
    var hasLoaded = false;
    function l() { if (hasLoaded) { return; } hasLoaded = true; window.doorbellOptions.windowLoaded = true; var g = d.createElement(t); g.id = 'doorbellScript'; g.type = 'text/javascript'; g.async = true; g.src = 'https://embed.doorbell.io/button/' + window.doorbellOptions['id'] + '?t=' + (new Date().getTime()); (d.getElementsByTagName('head')[0] || d.getElementsByTagName('body')[0]).appendChild(g); }
    if (w.attachEvent) { w.attachEvent('onload', l); } else if (w.addEventListener) { w.addEventListener('load', l, false); } else { l(); }
    if (d.readyState === 'complete') { l(); }
  }(window, document, 'script'))
  return true
}

let doorbell = window.doorbell

// have to wait until doorbell's script loads
const awaitDoorbell = method => params => {
  if (doorbell === undefined) {
    doorbell = window.doorbell
  }
  if (doorbell === undefined) {
    setTimeout(function () { awaitDoorbell(method)(params) }, 100)
  } else {
    method(params)
  }
}

export const identity = {
  email: "",
  name: "",
}

const identify = ({ email, name, id }) => {
  identity.email = email
  identity.name = name

  doorbell.setOption("hideEmail", true)
  doorbell.setOption("email", email)
  doorbell.setProperty("name", name)
  doorbell.setProperty("id", id)
}
const wrappedIdentify = awaitDoorbell(identify)

const deidentify = () => {
  identity.email = ""
  identity.name = ""

  doorbell.setOption("hideEmail", false)
  doorbell.setOption("email", "")
  doorbell.setProperty("name", "")
  doorbell.setProperty("id", "")
}
const wrappedDeidentify = awaitDoorbell(deidentify)

const show = (params) => {
  const showHandler = params && params.showHandler
  const hideHandler = params && params.hideHandler
  const successHandler = params && params.successHandler
  const tags = params && params.tags

  showHandler && doorbell.setOption("onShow", () => { showHandler() })
  hideHandler && doorbell.setOption("onHide", () => {
    hideHandler()
    reset()
  })
  successHandler && doorbell.setOption("onSuccess", () => { successHandler() })
  tags && doorbell.setOption("tags", "electron," + tags.toString())

  doorbell.show()
}
const wrappedShow = awaitDoorbell(show)

const hide = () => {
  doorbell.hide()
}
const wrappedHide = awaitDoorbell(hide)

const send = (params) => {
  const tags = params.tags ? params.tags : []
  doorbell.setOption("tags", "electron," + tags.toString())
  params.onSuccess && doorbell.setOption("onSuccess", params.onSuccess)
  params.onError && doorbell.setOption("onSuccess", params.onError)
  doorbell.send(params.message, params.email)
}
const wrappedSend = awaitDoorbell(send)

// Resets Doorbell titles and forms to default
const reset = () => {
  document.querySelectorAll("#doorbell-title").forEach((element, _) => {
    element.textContent = "Talk to us"
  })
  doorbell.setOption("onShow", () => { })
  doorbell.setOption("onHide", () => { })
  doorbell.setOption("onSuccess", () => { })
  doorbell.setOption("tags", "electron")
}

export {
  wrappedIdentify as identify,
  wrappedDeidentify as deidentify,
  wrappedShow as show,
  wrappedHide as hide,
  wrappedSend as send,
}
