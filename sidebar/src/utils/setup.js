import { localhostProxy } from './urls'
import * as analytics from "./analytics"
import * as cio from "./customer-io"
import * as mixpanel from "./mixpanel"

const electron = window.require("electron")

export const completionFlow = async ({
  setSetupCompleted,
  forceCheckOnline,
  getHaveShownWelcome,
  setHaveShownWelcome,
  metricsId,
  installId,
}) => {
  // An non-matching id indicates the user logged in and recieved an updated one.
  if (cio.getId() !== installId)
    registerOrphan(installId)
  setSetupCompleted()
  const { success, isOnline } = await forceCheckOnline()
  if (!(success && isOnline)) return
  const haveShown = await getHaveShownWelcome()
  if (haveShown) return
  electron.shell.openExternal(localhostProxy(`/clientapi/desktoplogin?d=/welcome${metricsId ? `?id=${metricsId}` : ''}`))
  setHaveShownWelcome()
  sendInstallCompletedEvent()
}

const registerOrphan = (orphanId) => {
  const currentId = analytics.get_distinct_id()
  mixpanel.identify(orphanId)
  mixpanel.people_set({ orphaned: true })
  mixpanel.identify(currentId)
}

const sendInstallCompletedEvent = () => {
  // Waiting on mixpanel alias from webapp. Not ideal for correctness but quick to implement.
  const SAFETY_INTERVAL_MS = 60 * 1000
  setTimeout(() => {
    analytics.track({ event: "kite_install_completed" })
  }, SAFETY_INTERVAL_MS)
}
