import React from 'react'
import { connect } from 'react-redux'

class StyleWatcher extends React.Component {

  useTheme = (theme, shouldTransition) => {
    if(shouldTransition) {
      document.documentElement.classList.add('in-color-theme-transition')
    }
    document.documentElement.setAttribute('data-color-theme', theme)
    if(shouldTransition) {
      setTimeout(() => {
        document.documentElement.classList.remove('in-color-theme-transition')
      }, 1000)
    }
  }

  useFont = (font) => {
    document.documentElement.setAttribute('data-font-theme', font.value)
  }

  useZero = (zero) => {
    document.documentElement.setAttribute('data-font-zero-type', zero)   
  }

  componentDidMount() {
    const {
      theme,
      font,
      zero
    } = this.props.style
    this.useTheme(theme, false)
    this.useFont(font)
    this.useZero(zero)
  }

  componentDidUpdate(prevProps) {
    const {
      theme,
      font,
      zero
    } = this.props.style
    if(theme !== prevProps.style.theme) {
      this.useTheme(theme, true)
    }
    if(font.value !== prevProps.style.font.value) {
      this.useFont(font)
    }
    if(zero !== prevProps.style.zero) {
      this.useZero(zero)
    }
  }

  render() {
    return null
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  style: state.stylePopup
})

const mapDispatchToProps = dispatch => ({

})

export default connect(mapStateToProps, mapDispatchToProps)(StyleWatcher)