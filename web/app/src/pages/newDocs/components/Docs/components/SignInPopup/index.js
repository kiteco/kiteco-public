import React from 'react'
import { connect } from 'react-redux'
import ReactDOM from 'react-dom'
import onClickOutside from 'react-onclickoutside'

import * as signinPopup from '../../../../../../redux/actions/signin-popup'
import SignIn from '../../../SignIn'

import './signin-popup.css'

class SignInPopup extends React.Component {

  handleClickOutside = e => {
    //dispatch toggle off action
    this.props.toggle(false)
  }

  componentDidUpdate() {
    if(this.props.signinPopup.visible) {
      ReactDOM.findDOMNode(this).scrollIntoView({
        behavior: 'smooth'
      })
    }
  }

  render() {
    const { visible } = this.props.signinPopup
    return (
      <div  className={`sign-in-popup ${visible ? 'visible' : ''}`}>
        <SignIn className='sign-in-popup' submitSuccessCb={this.handleClickOutside}/>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  signinPopup: state.signinPopup
})

const mapDispatchToProps = dispatch => ({
  toggle: (show) => dispatch(signinPopup.toggleSigninPopup(show))()
})

export default connect(mapStateToProps, mapDispatchToProps)(onClickOutside(SignInPopup))
