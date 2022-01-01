import React from 'react'
import ReactDOM from 'react-dom'
import SettingsHome from '../containers/SettingsHome'

it('renders the homepage without crashing', () => {
  const div = document.createElement('div')
  ReactDOM.render(<SettingsHome />, div)
})
