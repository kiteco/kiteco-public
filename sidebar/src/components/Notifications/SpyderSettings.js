import React from 'react'
import { connect } from 'react-redux'
import { setHasSeenSpyderNotification, setSpyderOptimalSettings } from '../../actions/system'
import { Domains } from '../../utils/domains'

class SpyderSettings extends React.Component {
  constructor(props) {
    super(props)
  }

  render() {
    const { runningEditor, setSpyderOptimalSettings, setHasSeenSpyderNotification, dismiss } = this.props

    let clickHandler = () => {
      dismiss()
      setHasSeenSpyderNotification()
      setSpyderOptimalSettings()
    }

    let hideHandler = () => {
      dismiss()
      setHasSeenSpyderNotification()
    }

    return <div className="notifications__spyder">
      <div className="notifications__spyder__header">
        <div className="notifications__spyder__title">Change your Spyder settings</div>
        <div className="notifications__spyder__hide" onClick={hideHandler}>Hide</div>
      </div>
      <div className="notifications__spyder__content">
        <div className="notifications__spyder__p">
          Kite works best in Spyder if you change autocompletions to show up after 1 character and after 100ms or less.
          <a className="notifications__spyder__a"
            target="_blank"
            rel="noopener noreferrer"
            href={`https://${Domains.Help}/article/90-using-the-spyder-plugin#spyder-setup`}>Learn more</a>
        </div>

        { !runningEditor &&
          <a className="notifications__spyder__apply" onClick={clickHandler}>Change settings for me</a>
        }
      </div>
    </div>
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
  setSpyderOptimalSettings: () => dispatch(setSpyderOptimalSettings()),
  setHasSeenSpyderNotification: () => dispatch(setHasSeenSpyderNotification()),
})

export default connect(mapStateToProps, mapDispatchToProps)(SpyderSettings)
