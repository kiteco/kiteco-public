// isEnterprise returns if the app is currently being run in
// enterprise
const isEnterprise = () => {
  if (process.env.REACT_APP_ENTERPRISE === "1") {
    return true
  }
  return window.KITE_ENTERPRISE === "true"
}

export const ENTERPRISE = isEnterprise()
