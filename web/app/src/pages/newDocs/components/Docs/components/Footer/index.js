import React from 'react'
import { connect } from 'react-redux'

import CommonFooter from '../../../../../../components/Footer'

import './footer.css'

import { toggleSigninPopup } from '../../../../../../redux/actions/signin-popup'

class Footer extends React.Component {
  render() {
    const { toggleSignin, loggedIn } = this.props
    return (
      <footer className="documentation__footer">
        <CommonFooter/>
        { !loggedIn &&
          <button onClick={toggleSignin(true)} className='subtle'>Sign In</button>
        }
      </footer>
    )
  }
}


const mapStateToProps = (state, ownProps) => ({
  loggedIn: state.account.status === "logged-in",
})

const mapDispatchToProps = dispatch => ({
  toggleSignin: (show) => dispatch(toggleSigninPopup(show))
})


export default connect(mapStateToProps, mapDispatchToProps)(Footer)
