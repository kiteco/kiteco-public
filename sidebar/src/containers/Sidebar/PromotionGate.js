import React from 'react'
import { connect } from 'react-redux'

import { navigatorOs } from '../../utils/navigator'

import * as promotions from '../../actions/promotions'

import './assets/promotion.css'

const PromotionGate = ({
   views,
   loggedIn,
   resetDocsViews,
   children,
   os,
}) => {
  return <div className="docs__promotion-wrapper">
    {React.Children.only(children)}
  </div>
}

const mapStateToProps = (state, ownProps) => ({
  views: state.promotions.docsViews,
  loggedIn: state.account.status === "logged-in",
  os: ownProps.os || navigatorOs(),
})

const mapDispatchToProps = dispatch => ({
  resetDocsViews: () => dispatch({ type: promotions.RESET_DOCS_VIEWS }),
})

export default connect(mapStateToProps, mapDispatchToProps)(PromotionGate)
