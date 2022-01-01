import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'

import './assets/plan-gate.css'

/**
 * PlanGate acts as a wrapper for elements that
 * should only be available for users with
 * access to certain feature sets.
 *
 * For now, this container only checks to see if
 * a required `feature` is set to true in `features`
 * of the `account.plan` store.
 */
const PlanGate = ({
  requiredFeature,
  featureDescription,
  features,
  startedTrial,
  startTrial,
  children,
}) => {

  if (features[requiredFeature]) {

    return React.Children.only(children)

  } else {

    return <div className="plan-gate">
      <div className="plan-gate__pro-logo"/>
      { featureDescription ?
        <div>{featureDescription}</div>
        : <div>This is a Kite Pro feature.</div>
      }
      { !startedTrial ?
        <div> Start your Kite Pro Trial today! </div>
        : <div>
          Your Kite Pro trial has ended. Sign up for Kite Pro today!
        </div>
      }
      <div>
        { false && !startedTrial &&
          <button
            onClick={startTrial}
            className="plan-gate__call-to-action--button"
          >
            Start Trial
          </button>
        }
        <Link
          to="/pro"
          className="plan-gate__call-to-action--link"
        >
          Learn more
        </Link>
      </div>
    </div>

  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  features: state.account.plan.features || {},
  startedTrial: state.account.plan.started_kite_pro_trial,
})

const mapDispatchToProps = dispatch => ({
  // startTrial: () => dispatch(actions.startTrialPlan()),
})

export default connect(mapStateToProps, mapDispatchToProps)(PlanGate)
