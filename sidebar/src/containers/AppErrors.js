import React from 'react'
import { connect } from 'react-redux'
import ErrorOverlay from '../components/ErrorOverlay'
import { Domains } from '../utils/domains'

class AppErrors extends React.Component {
  hasHadPollingActionTaken() {
    return this.props.polling.pollingSuccess || this.props.polling.restartSuccess ||
            this.props.polling.restartError || this.props.polling.attemptRestart || this.props.polling.noSupport
  }

  isApplicationError(pollingActionTaken) {
    return (
      this.props.errors &&
      !this.props.errors.online &&
      !pollingActionTaken
    ) ||
    (
      this.props.errors &&
      !this.props.errors.responsive &&
      this.props.errors.online &&
      !pollingActionTaken
    ) ||
    (
      this.props.polling &&
      this.props.polling.pollingSuccess
    ) ||
    (
      this.props.polling &&
      this.props.polling.attemptRestart
    ) ||
    (
      this.props.polling &&
      this.props.polling.restartSuccess
    ) ||
    (
      this.props.polling &&
      this.props.polling.restartError
    ) ||
    (
      this.props.polling &&
      this.props.polling.noSupport
    )
  }

  render() {
    const pollingActionTaken = this.hasHadPollingActionTaken()
    return (
      <div>
        { this.props.errors &&
          !this.props.errors.online &&
          !pollingActionTaken &&
          <ErrorOverlay
            title="Kite engine is not running"
            subtitle={`We're polling to see if it'll come back online. After a few attempts, we'll try to restart. You can also relaunch Kite now`}
            handler={this.props.reloadHandler}
            btnText="Launch Kite"
            isSeeThrough={true}
          />
        }
        { this.props.errors &&
          !this.props.errors.responsive &&
          this.props.errors.online &&
          !pollingActionTaken &&
          <ErrorOverlay
            title="Kite engine is unresponsive"
            subtitle="Would you like to restart Kite? We'll keep trying to see if we can get a response"
            handler={this.props.reloadHandler}
            btnText="Restart Kite"
            isSeeThrough={true}
          />
        }
        { this.props.polling &&
          this.props.polling.pollingSuccess &&
          <ErrorOverlay
            title="Good News, Everyone!"
            subtitle="We're back online and refreshing momentarily"
            spinner={true}
          />
        }
        { this.props.polling &&
          this.props.polling.attemptRestart &&
          <ErrorOverlay
            title="Here We Go"
            subtitle="We're attempting to restart Kite Engine"
            spinner={true}
          />
        }
        { this.props.polling &&
          this.props.polling.restartSuccess &&
          <ErrorOverlay
            title="Kite was restarted successfully!"
            subtitle="Blast off coming"
            spinner={true}
          />
        }
        { this.props.polling &&
          this.props.polling.restartError &&
          <ErrorOverlay
            title="We're having some trouble restarting Kite..."
            subtitle="You can try again or visit Kite.com to reinstall and start fresh"
            linkText="Reinstall"
            link={`https://${Domains.PrimaryHost}/download`}
            isSeeThrough={true}
            handler={this.props.reloadHandler}
            btnText="Try Again"
          />
        }
        { this.props.polling &&
          this.props.polling.noSupport &&
          <ErrorOverlay
            title="Kite isn't supported on your OS"
            subtitle="Please open a GitHub issue if you believe this is an error"
            linkText="Open issue"
            link="https://github.com/kiteco/issue-tracker"
            isSeeThrough={true}
          />
        }
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  errors: state.errors,
  polling: state.polling,
  system: state.system,
  status: state.account.status,
})

export default connect(mapStateToProps)(AppErrors)
