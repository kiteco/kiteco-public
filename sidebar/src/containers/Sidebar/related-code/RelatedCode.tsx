import * as React from "react"
import { connect } from "react-redux"

import { Status } from "../../../store/related-code/related-code"

import RelatedCodeHeader from "./header/RelatedCodeHeader"
import RelatedCodeHome from "./home/RelatedCodeHome"
import RelatedCodeSearch from "./search/RelatedCodeSearch"

import './related-code-index.css'
import RCHeader from "../../RemoteContent/RCHeader";

/*
  RelatedCode is the top-level component for the Related Code Dashboard
 */
class RelatedCode extends React.Component<any, any> {
  render() {
    const { os } = this.props
    const { related_code: { status }} = this.props

    let pageContent
    if (status === Status.NoSearch) {
      pageContent = <RelatedCodeHome />
    } else {
      pageContent = <RelatedCodeSearch />
    }

    return (
      <div className={ `wrapper-page wrapper-page--${os}` }>
        <RCHeader />
        <RelatedCodeHeader />
        <div className="related-code-page">
          {pageContent}
        </div>
      </div>
    )
  }
}

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    os: state.system.os,
    related_code: state.related_code,
  }
}

export default connect(mapStateToProps, null)(RelatedCode)
