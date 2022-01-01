import React, { useState } from 'react'
import { connect } from 'react-redux'
import { Link, NavLink, Route, Switch } from 'react-router-dom'
import { Redirect } from "react-router"
import { AnyAction } from "redux"
import { ThunkDispatch } from "redux-thunk"

import Notifications from '../../components/Notifications'
import ProductBadge from '../../components/ProductBadge'
import TopBar from '../../components/TopBar'
import WindowMode from '../../components/WindowMode'

import Docs from './docs/Docs'
import Examples from '../Examples'
import RelatedCode from "./related-code/RelatedCode"

import './assets/sidebar-common.css'
import './docs/docs-index.css'
import styles from './index.module.css'

import { getInstalledPlugins, isAnyEditorInstalled } from '../../utils/plugins'

import { getAutosearchDefault } from '../../actions/settings'
import { getUser } from '../../actions/account'
import { disableAutosearch, enableAutosearch } from '../../actions/search'
import { Notification, NotifType, notify } from '../../store/notification'
import { getSpyderOptimalSettings } from '../../actions/system'
import { getPlugins, resetAutoInstalledPlugins } from '../../actions/plugins'

interface SidebarProps {
  notify: (params: Notification) => any,
  getUser: () => any,
  getPlugins: () => any,
  resetAutoInstalledPlugins: () => any,
  getAutosearchDefault: () => any,
  enableAutosearch: () => any,
  disableAutosearch: () => any,
  getSpyderOptimalSettings: () => any,
  autosearchDefault: boolean,
  setupCompleted: boolean | string,
  os: any,
  hasSeenSpyderNotification: boolean,
  shouldBlur: boolean,
}

class Sidebar extends React.Component<SidebarProps> {
  componentDidMount() {
    const {
      autosearchDefault,
      getAutosearchDefault,
      enableAutosearch,
      disableAutosearch,
      notify,
      setupCompleted,
      getUser,
      getPlugins,
      resetAutoInstalledPlugins,
    } = this.props

    if (autosearchDefault === null) {
      getAutosearchDefault()
        .then((resp: any) => {
          if (resp.success) {
            if (resp.data) {
              enableAutosearch()
            } else {
              disableAutosearch()
            }
          }
        })
    }

    getUser()
    if (setupCompleted === true || setupCompleted === 'true') {
      const pluginNotify = () => {
        setTimeout(() => {
          notify({
            id: 'noplugins',
            component: NotifType.NoPlugins,
            docsOnly: false,
          })
        }, 500)
      }
      getPlugins().then((resp: any) => {
        if (resp && resp.success) {
          const { data } = resp
          const installedPlugins = getInstalledPlugins(data.plugins)
          if (installedPlugins.length === 0 && isAnyEditorInstalled(data.plugins)) {
            pluginNotify()
          }
        } else {
          pluginNotify()
        }
      })

      resetAutoInstalledPlugins().then((resp: any) => {
        if (resp && resp.success) {
          notify({
            id: 'plugin-auto-installed',
            component: NotifType.PluginsAutoInstalled,
            payload: {
              pluginIDs: resp.data,
            },
            docsOnly: false,
          })
        }
      })

      // TODO(Daniel): Figure out how to streamline notifications. Taking this
      // out while we launch Kite Pro...
      //
      // if(!hasSeenSpyderNotification) {
      //   getSpyderOptimalSettings().then(resp => {
      //     if(resp && resp.success && resp.data && resp.data.optimalSettings === false) {
      //       notify({
      //         id: 'spyder-settings',
      //         component: 'spyder-settings',
      //         payload: resp.data
      //       })
      //     }
      //   })
      // }
    }
  }

  render() {
    const { os } = this.props

    return <div className={`main docs ${this.props.shouldBlur ? 'main--blur' : ''}`}>
      {os !== 'windows' && <TopBar/>}
      <Switch>
        <Route path="/docs"
          component={Docs}
        />
        <Route path="/examples"
          component={Examples}
        />
        <Route path="/home">
          <Redirect to="/docs"/>
        </Route>
        <Route path="/related-code"
          component={RelatedCode}
        />
      </Switch>

      <Switch>
        <Route path="/related-code">
          <Notifications hideDocsNotifications={true}/>
        </Route>
        <Route>
          <Notifications/>
        </Route>
      </Switch>
      <div className="sidebar__tray">
        <ProductBadge className={styles.product_badge}/>
        <div className="sidebar__pages">
          <TrayTooltip
            text="View documentation for Python packages and your own Python code."
            title="Python Documentation"
            extraThin={true}
          >
            <NavLink
              activeClassName="active"
              className="sidebar__icon__docs"
              to="/docs"
            />
          </TrayTooltip>
          <TrayTooltip
            text="Navigate your codebase to quickly find similar pieces of code."
            title="Related Code Finder"
            extraThin={true}
          >
            <NavLink
              activeClassName="active"
              className="sidebar__icon__related-code"
              to="/related-code"
            />
          </TrayTooltip>
          <div className="sidebar__line"/>
          <WindowMode/>
          <Link className="sidebar__icon__settings" to="/settings"/>
          <TrayTooltip
            text="The software behind Kite has been acquired by another company. In this period of transition, you can continue to use any Kite software you possess, but neither we nor the new owner will be offering the software for download or supporting it at this time. We will explore new paths forward for our users to come. Thank you for your support of Kite, and please stay tuned for updates!"
            title="Notice for users"
            extraThin={false}
          >
            <div className="sidebar__icon__notice"/>
          </TrayTooltip>
        </div>
      </div>
    </div>
  }
}

interface TrayTooltipProps {
  children?: any,
  text: string,
  title: string,
  extraThin: boolean,
}

const TrayTooltip = (props: TrayTooltipProps) => {
  const [hovering, setHovering] = useState(false)
  return (
    <div
      className="sidebar__tooltip-wrapper"
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      {props.children}
      {hovering &&
      <div className={`sidebar__tooltip ${props.extraThin && "sidebar__tooltip--extra-thin"}`}>
        <div className="sidebar__tooltip__title">
          {props.title}
        </div>
        <p className="sidebar__tooltip__paragraph">
          {props.text}
        </p>
      </div>
      }
    </div>
  )
}

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  notify: (params: Notification) => dispatch(notify(params)),
  getUser: () => dispatch(getUser()),
  getPlugins: () => dispatch(getPlugins()),
  resetAutoInstalledPlugins: () => dispatch(resetAutoInstalledPlugins()),
  getAutosearchDefault: () => dispatch(getAutosearchDefault()),
  enableAutosearch: () => dispatch(enableAutosearch()),
  disableAutosearch: () => dispatch(disableAutosearch()),
  getSpyderOptimalSettings: () => dispatch(getSpyderOptimalSettings()),
})

const mapStateToProps = (state: any) => ({
  autosearchDefault: state.settings.autosearchDefault,
  setupCompleted: state.settings.setupCompleted,
  os: state.system.os,
  hasSeenSpyderNotification: state.system.hasSeenSpyderNotification,
})

export default connect(mapStateToProps, mapDispatchToProps)(Sidebar)
