import * as React from "react"
import { connect } from "react-redux"
import { push } from "react-router-redux"
import { ThunkDispatch } from "redux-thunk"
import { AnyAction } from "redux"

/*
	RelatedCodeRedirector listens for a new Related Code search and redirects to the Related Code dashboard
 */
class RelatedCodeRedirector extends React.Component<any, any> {
  componentDidUpdate() {
    const { id: nextId, push } = this.props
    if (nextId) {
      push("/related-code")
    }
  }

  render() {
    return null
  }
}

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  push: (params: any) => dispatch(push(params)),
})

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    id: state.related_code.id,
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(RelatedCodeRedirector)
