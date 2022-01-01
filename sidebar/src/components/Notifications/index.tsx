import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from "redux"
import { ThunkDispatch } from "redux-thunk"

import Default from './Default'
import Autosearch from './Autosearch'
import Plugins from './Plugins'
import RemotePluginOffline from './RemotePluginOffline'
import PluginsAutoInstalled from './PluginsAutoInstalled'
import RunningPluginInstallFailure from './RunningPluginInstallFailure'
import NoPlugins from './NoPlugins'
import Offline from './Offline'
import SpyderSettings from './SpyderSettings'

import { dismiss, Notification, NotifType } from '../../store/notification'

import './notifications.css'

const componentsMap: Record<NotifType, any> = {
  [NotifType.Default]: Default,
  [NotifType.Autosearch]: Autosearch,
  [NotifType.NoPlugins]: NoPlugins,
  [NotifType.Plugins]: Plugins,
  [NotifType.Offline]: Offline,
  [NotifType.PluginsAutoInstalled]: PluginsAutoInstalled,
  [NotifType.RemotePluginOffline]: RemotePluginOffline,
  [NotifType.RunningPluginInstallFailure]: RunningPluginInstallFailure,
  [NotifType.SpyderSettings]: SpyderSettings,
}

interface NotificationsProps {
  hideDocsNotifications: boolean,
  notifications: Notification[],
  dismiss: any,
}

const Notifications = (props: NotificationsProps) =>
  <div className="notifications">
    { props.notifications.map(n => {
      const { component, payload, id, docsOnly } = n
      const Component = component ? componentsMap[component] : Default
      if (docsOnly && props.hideDocsNotifications) {
        return null
      }
      return <Component
        { ...payload }
        key={id}
        dismiss={() => props.dismiss(id)}
      />
    })}
  </div>

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  dismiss: (id: any) => dispatch(dismiss(id)),
})

const mapStateToProps = (state: any) => ({
  notifications: state.notification,
})

export default connect(mapStateToProps, mapDispatchToProps)(Notifications)
