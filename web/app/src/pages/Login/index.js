import React from 'react'
import Helmet from 'react-helmet'

import ScrollToTop from '../../components/ScrollToTop'
import SignInForm from './SignInForm'
import Header from '../../components/Header'

import './assets/login.css'

const Login = ({ headerType="root", pro=false }) =>
  <div className="login">
    <ScrollToTop/>
    <Helmet>
      <title>Kite Login</title>
    </Helmet>
    <Header type={headerType}/>
    <div className="login__wrapper">
      <SignInForm />
    </div>
  </div>

export default Login
