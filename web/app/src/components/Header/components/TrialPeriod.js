import React from 'react'
import { Link } from 'react-router-dom'

const TrialPeriod = ({
  started_kite_pro_trial,
  trial_days_remaining,
  active_subscription,
  status,
}) => {
  if (
    status === "trialing" &&
    trial_days_remaining < 7
  ) {
    return <div
      className="header__trial"
    >
      {`You have ${trial_days_remaining}
      trial day${trial_days_remaining === 1 ? "" : "s"}
      remaining. `}
      <Link
        className="header__trial__link"
        to="/pro"
      >
        Learn more
      </Link>
    </div>
  }
  if (active_subscription === "pro") {
    return null
  }
  if (
    started_kite_pro_trial &&
    trial_days_remaining === 0
  ) {
    return <div
      className="header__trial"
    >
      Your Kite Pro trial has expired.
      <Link
        className="header__trial__link"
        to="/pro"
      >
        Learn more
      </Link>
    </div>
  }
  if (
    !started_kite_pro_trial &&
    trial_days_remaining !== 0
  ) {
    return <div
      className="header__trial"
    >
      {`Start your ${trial_days_remaining} day Kite Pro trial today! `}
      <Link
        className="header__trial__link"
        to="/trial"
      >
        Start Trial
      </Link>
    </div>
  }
  return null
}

export default TrialPeriod
