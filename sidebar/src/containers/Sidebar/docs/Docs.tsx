import * as React from 'react'
import { connect } from 'react-redux'
import { Route, Switch } from 'react-router-dom'
import { push } from 'react-router-redux'
import { ThunkDispatch } from "redux-thunk"
import { AnyAction } from "redux"

import { GET } from '../../../actions/fetch'

import Page from './Page'
import DocsHome from './DocsHome'

import Autosearch from "../../../components/Autosearch/index"
import Search from "../../../components/Search/index"
import RCHeader from "../../RemoteContent/RCHeader"

import './docs-index.css'

const Docs = (props: any) => (
  <div className={ `wrapper-page wrapper-page--${props.os}` }>
    <RCHeader />
    <Autosearch />
    <div className="docs-page__search">
      <Search
        get={props.get}
        push={props.push}
      />
    </div>
    <div className="docs-page">
      <Switch>
        <Route path={`${props.url}/:id`}
          component={Page}
        />
        { /* Render this for all other docs paths including root */ }
        <Route render={() =>
          <DocsHome identifier={props.docs.identifier} />
        }/>
      </Switch>
    </div>
  </div>
)

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  get: (params: any) => dispatch(GET(params)),
  push: (params: any) => dispatch(push(params)),
})

const mapStateToProps = (state: any, ownProps: any) => ({
  url: ownProps.match.url,
  docs: state.docs,
  os: state.system.os,
})

export default connect(mapStateToProps, mapDispatchToProps)(Docs)
