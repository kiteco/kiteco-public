import * as React from 'react'
import { connect } from 'react-redux'
import { RemoteContentState } from '../../store/remotecontent'

const electron = window.require('electron')

interface RCDocsDashboardParagraphProps {
  remotecontent: RemoteContentState
}

class RCDocsDashboardParagraph extends React.Component<RCDocsDashboardParagraphProps> {
  render() {
    const item = this.props.remotecontent.docs_dashboard_paragraph
    if (!item || !item.content) {
      return null
    }
    if (item.link) {
      return <a
        style={{ cursor:'pointer' }}
        onClick={() => electron.shell.openExternal(item.link)}
        dangerouslySetInnerHTML={{ __html: item.content }}
      />
    }
    return <div dangerouslySetInnerHTML={{ __html: item.content }} />
  }
}

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    remotecontent: state.remotecontent,
  }
}

export default connect(mapStateToProps, null)(RCDocsDashboardParagraph)
