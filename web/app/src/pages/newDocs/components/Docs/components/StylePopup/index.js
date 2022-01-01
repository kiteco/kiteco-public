import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import onClickOutside from 'react-onclickoutside'
import * as stylePopup from '../../../../../../redux/actions/style-popup'
import constants from '../../../../../../utils/theme-constants'

const { FONTS, THEMES, ZEROES } = constants

const ColorThemeItem = ({theme, onThemeChange, currentTheme}) => {
  return (<li>
    <label className='color-theme'>
      <input 
        type='radio' 
        name='color-theme' 
        value={theme} 
        checked={theme === currentTheme} 
        className={`color-theme-preview-${theme}`}
        onChange={onThemeChange} />
      <span className='label'>{theme}</span>
    </label>
  </li>)
}

const FontItem = ({font, onFontChange, currentFont}) => {
  return (<li>
    <label>
      <input 
        type='radio' 
        name='font-theme' 
        value={font.value} 
        checked={currentFont.value === font.value}
        onChange={onFontChange} />
      <span className='label' data-forced-font-theme={font.value}>{font.name}</span>
      <span className='hint'>{font.description}</span>
    </label>
  </li>)
}

const ZeroItem = ({zero, onZeroChange, currentZero}) => {
  return (<li>
    <label>
      <input 
        type='radio' 
        name='font-zero-type' 
        value={zero} 
        checked={zero === currentZero}
        onChange={onZeroChange} />
      <span className='label'><code className='code'
          data-forced-font-zero-type={zero}>[0, 1]</code></span>
      <span className='hint'>{zero} zero</span>
    </label>
  </li>)
}

class StylePopup extends React.Component {

  handleClickOutside = evt => {
    //dispatch toggle off action
    this.props.toggle(false)
  }

  componentDidUpdate() {
    if(this.props.stylePopup.visible) {
      ReactDOM.findDOMNode(this).scrollIntoView({
        behavior: 'smooth'
      })
    }
  }

  render() {
    const {
      visible,
      theme: currentTheme,
      font: currentFont,
      zero: currentZero,
    } = this.props.stylePopup
    const {changeTheme, changeFont, changeZero} = this.props
    return (
      <div className={`style-popup ${visible ? 'visible' : ''}`}>
        <div className='options'>
          <div>
            <ul className='color-themes'>
              {Object.keys(THEMES).map(key => <ColorThemeItem 
                key={key}
                theme={THEMES[key]}
                currentTheme={currentTheme} 
                onThemeChange={changeTheme}/>)}
            </ul>
          </div>
          <div>
            <ul>
              {FONTS.map(font => <FontItem 
                key={font.value}
                font={font}
                currentFont={currentFont} 
                onFontChange={changeFont}/>)}
            </ul>
          </div>
          <div>
            <ul>
              {Object.keys(ZEROES).filter(key => {
                if(currentFont.zeroes[ZEROES[key]]) {
                  return true
                }
                return false
              }).map(key => <ZeroItem
                  key={key} 
                  zero={ZEROES[key]} 
                  currentZero={currentZero}
                  onZeroChange={changeZero}/>)}
            </ul>
          </div>
        </div>
      </div>
    )
  }
}

const mapDispatchToProps = dispatch => ({
  toggle: (show) => dispatch(stylePopup.toggleStylePopup(show))(),
  changeTheme: (evt) => dispatch(stylePopup.changeTheme(evt.currentTarget.value)),
  changeFont: (evt) => dispatch(stylePopup.changeFont(evt.currentTarget.value)),
  changeZero: (evt) => dispatch(stylePopup.changeZero(evt.currentTarget.value))
})

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  stylePopup: state.stylePopup,
})

export default connect(mapStateToProps, mapDispatchToProps)(onClickOutside(StylePopup))
