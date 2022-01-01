import React from 'react'
import { connect } from 'react-redux'
import Animated from "animated/lib/targets/react-dom"
import { TwitterShareButton, LinkedinShareButton, FacebookShareButton } from 'react-share'
import SocialSvg from './SocialSvg'

import './style.css'

const getLastCompletionText = (lastCompletion="a line") => `Wow. I just test drove Line-of-Code Completions from Kite and it completed \`${lastCompletion}\` for me in one go. This is mind-blowing. 🤯  Try it out in your browser.`

const scrubEmoji = (text) => {
  return text.replace('🤯', '')
}

class SandboxSocialSharing extends React.Component {
  constructor(props) {
    super(props)

    let sharingText = props.lastCompletion
      ? getLastCompletionText(props.lastCompletion)
      : "Wow. I just test drove Line-of-Code Completions from Kite. This is mind-blowing. 🤯 Try it out in your browser."

    let opacityAnim = new Animated.Value(1)
    let heightAnim = new Animated.Value(1)

    this.state = {
      expanded: true,
      userHasEditedSharingText: false,
      highlighted: '',
      relativeLineCount: 0,
      initialLineCount: 0,
      baseMarginTop: 0,
      opacityAnim,
      heightAnim,
      sharingText,
    }
  }

  componentDidUpdate(prevProps) {
    // doesn't match and hasn't been edited
    if(!this.state.userHasEditedSharingText && prevProps.lastCompletion !== this.props.lastCompletion) {
      if(!this.props.lastCompletion) {
        this.setState({ sharingText:  "Wow. I just test drove Line-of-Code Completions from Kite. This is mind-blowing. 🤯 Try it out in your browser." })
      }
      this.setState({ sharingText: getLastCompletionText(this.props.lastCompletion)})
    }

    if(prevProps.shouldCollapse !== this.props.shouldCollapse) {
      if(this.props.shouldCollapse) {
        this.collapse()
      } else {
        this.expand()
      }
    }

    if(prevProps.editingInitialized !== this.props.editingInitialized && this.props.editingInitialized) {
      this.setState({ initialLineCount: this.props.lineCount })
    }

    if(prevProps.editingInitialized && this.props.editingInitialized && prevProps.lineCount !== this.props.lineCount) {
      const lineDiff = this.props.lineCount - prevProps.lineCount
      this.setState({ relativeLineCount: this.state.relativeLineCount + lineDiff })
    }
  }

  componentDidMount(prevProps) {
    const computed = window.getComputedStyle(this.socialContainer)
    this.setState({ baseMarginTop: parseInt(computed.getPropertyValue('margin-top'), 10) })
  }

  buttonFillFromTheme = (theme, type) => {
    switch(theme) {
      case 'kite-light':
        if(this.state.highlighted === type) {
          return '#2f76ce'
        }
        return '#5a91d4' //--sandbox-social-btn-color
      case 'kite-dark':
        if(this.state.highlighted === type) {
          return 'rgb(30, 139, 255)'
        }
        return 'rgb(54, 105, 160)'
      default:
        return ''
    }
  }

  backgroundFromTheme = (theme) => {
    switch(theme) {
      case 'kite-light':
        return '#f5f5f5'
      case 'kite-dark':
        return '#1e1e1e'
      default:
        return ''
    }
  }

  handleTextChange = (event) => {
    this.setState({ sharingText: event.target.value })
  }

  handleShareBtnMouseEnter = (type) => (event) => {
    event.stopPropagation()
    if(this.state.highlighted !== type) {
      this.setState({ highlighted: type })
    }
  }

  handleShareBtnMouseLeave = (type) => (event) => {
    event.stopPropagation()
    if(this.state.highlighted === type) {
      this.setState({ highlighted: "" })
    }
  }

  collapse = () => {
    this.setState(() => {
      Animated.sequence(
        Animated.timing(this.state.heightAnim, { 
          toValue: 0, 
          duration: 300 
        }).start(),
        Animated.timing(this.state.opacityAnim, { 
          toValue: 0, 
          duration: 300 
        }).start()
      )
      return { expanded: false }
    })
  }

  expand = () => {
    this.setState(() => {
      Animated.sequence(
        Animated.timing(this.state.heightAnim, { 
          toValue: 1, 
          duration: 300 
        }).start(),
        Animated.timing(this.state.opacityAnim, { 
          toValue: 1, 
          duration: 300 
        }).start()
      )
      return { expanded: true }
    })
  }

  getSocialContainerStyle = () => {
    if(this.socialContainer) {
      const computed = window.getComputedStyle(this.socialContainer)
      const lineHeight =  this.props.codeLineHeight || parseInt(computed.getPropertyValue("line-height"), 10)
      return {
        marginTop: `${this.state.baseMarginTop + lineHeight * this.state.relativeLineCount}px`
      }
    }
    return {}
  }

  render() {
    const { theme, location } = this.props
    let { sharingText, expanded, opacityAnim, heightAnim } = this.state

    
    const postUrl = `https://${window.location.hostname}${location.pathname}`

    // dynamically compute margin-top based on relativeLineCount
    const containerStyle = this.getSocialContainerStyle()
    return <div
      className={`sandbox__social-sharing${theme ? ` ${theme}`: ''}`}
      ref={elem => this.socialContainer = elem}
      style={containerStyle}
    >
      <Animated.div 
        className={`sandbox__social-sharing__collapsible`}
        style={{opacity: opacityAnim, height: heightAnim.interpolate({
          inputRange: [0, 1],
          outputRange: ['0%', '100%']
        })}}
      >
        <div className="sandbox__social-sharing__textfield">
          <textarea 
            className="sandbox__social-sharing__textfield--area"
            value={sharingText}
            onChange={this.handleTextChange}
          />
        </div>
      </Animated.div>
      <div className="sandbox__social-sharing__container">
        <div 
          className="sandbox__social-sharing__buttons"
        >
          <div className="sandbox__social-sharing__textfield--title">
            Share this:
          </div>
          <div 
            className="sandbox__social-sharing__button"
            onMouseEnter={this.handleShareBtnMouseEnter('twitter')}
            onMouseLeave={this.handleShareBtnMouseLeave('twitter')}
          >
            <TwitterShareButton
              children={<SocialSvg fill={this.buttonFillFromTheme(theme, 'twitter')} type="twitter" />}
              url={postUrl}
              title={sharingText}
            />
          </div>
          <div 
            className="sandbox__social-sharing__button"
            onMouseEnter={this.handleShareBtnMouseEnter('facebook')}
            onMouseLeave={this.handleShareBtnMouseLeave('facebook')}
          >
          {/* Facebook quotes do not support the showing of emoji chars */}
            <FacebookShareButton
              children={<SocialSvg fill={this.buttonFillFromTheme(theme, 'facebook')} background={this.backgroundFromTheme(theme)} type="facebook" />}
              url={postUrl}
              quote={scrubEmoji(sharingText)}
            />
          </div>
          <div 
            className="sandbox__social-sharing__button"
            onMouseEnter={this.handleShareBtnMouseEnter('linked-in')}
            onMouseLeave={this.handleShareBtnMouseLeave('linked-in')}
          >
            <LinkedinShareButton
              children={<SocialSvg fill={this.buttonFillFromTheme(theme, 'linked-in')} type="linked-in" />}
              url={postUrl}
              title={sharingText}
            />
          </div>
        </div>
        <div className="sandbox__social-sharing__collapse-btn" onClick={expanded ? this.collapse : this.expand}>
          {expanded ? '⋏' : '⋎'}
        </div>
      </div>
    </div>
  }
  
}

const mapStateToProps = (state, props) => ({
  location: state.routing.location,
})

export default connect(mapStateToProps, null)(SandboxSocialSharing)