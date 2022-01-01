import React from 'react'
import { connect } from 'react-redux'

import { enableAutosearch } from '../../actions/search'
import { GET } from '../../actions/fetch'
import { symbolReportPath } from '../../utils/urls'
import { normalizeValueReportFromSymbol } from '../../utils/value-report'

class Autosearch extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      name: null,
    }
  }

  componentDidMount() {
    this.loadName()
  }

  componentDidUpdate(props) {
    const { id: nextId } = this.props
    const { id: prevId } = props
    if (prevId !== nextId && nextId !== "") {
      this.loadName(nextId)
    }
  }

  loadName = queryId => {
    const { id, get } = this.props
    queryId = queryId || id
    get({
      url: symbolReportPath(queryId),
      skipStore: true,
    }).then(({ success, data }) => {
      if (success && data) {
        this.setState({
          name: normalizeValueReportFromSymbol(data).value.repr,
        })
      }
    })
  }

  render() {
    const { enable, dismiss } = this.props
    const { name } = this.state
    if (name === null) {
      return null
    }
    return <div className="notifications__autosearch">
      <div className="notifications__autosearch__header">
        <div className="notifications__autosearch__title">
          Docs available
        </div>
        <div className="notifications__autosearch__hide"
          onClick={dismiss}
        >
          Hide
        </div>
      </div>
      <div className="notifications__autosearch__content">
        <div className="notifications__autosearch__identifier">
          { name }
        </div>
        <div
          className="notifications__autosearch__resume"
          onClick={enable}
        >
          Enable cursor following
        </div>
      </div>
    </div>
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  id: state.search.id,
})

const mapDispatchToProps = dispatch => ({
  enable: () => dispatch(enableAutosearch()),
  get: params => dispatch(GET(params)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Autosearch)
