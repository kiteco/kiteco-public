export const navigatorOs = () => {
  let operatingSystem = 'mac'

  const appVersion = navigator.appVersion || []
  const oscpu = navigator.oscpu || []

  if (appVersion.indexOf("Win")!==-1) {
    operatingSystem = 'windows'
  } else if (appVersion.indexOf("Mac")!==-1) {
    operatingSystem = 'mac'
  } else if (appVersion.indexOf("Linux")!==-1) {
    operatingSystem = 'linux'
  } else if (oscpu.indexOf("Linux")!==-1) {
    operatingSystem = 'linux'
  }

  return operatingSystem
}

export const isUseragentMobile = () => {
  return /(iPad|iPhone|iPod|Android)/g.test(navigator.userAgent)
}
