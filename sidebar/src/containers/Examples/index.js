import React from 'react'
import { Route, Redirect, Switch } from 'react-router-dom'

import Page from './Page'


const Examples = ({ match }) =>
  <div className="examples-page">
    <Switch>
      <Route path={`${match.url}/:language/:id`}
        component={Page}
      />
      <Redirect to="/docs"/>
    </Switch>
  </div>

export default Examples
