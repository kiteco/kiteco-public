import React from 'react'

const addTheme = (theme) => {
  document.documentElement.setAttribute('data-theme', theme)
}

class ColorTheme extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      currentTheme: this.props.theme
    }
  }

  componentDidMount() {
    addTheme(this.props.theme)
  }

  static getDerivedStateFromProps(props, state) {
    if(props.theme !== state.currentTheme) {
      addTheme(props.theme)
      return {
        currentTheme: props.theme 
      }
    }
    return null
  }

  render() {
    return this.props.children
  }
}

export default ColorTheme
