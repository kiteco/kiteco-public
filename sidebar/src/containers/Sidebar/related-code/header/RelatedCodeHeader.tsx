import * as React from "react"
import { connect } from "react-redux"
import { State, Status } from "../../../../store/related-code/related-code"

import styles from './related-code-header.module.css'
import { Domains } from "../../../../utils/domains"

const electron = window.require('electron')

type RelatedCodeHeaderProps = {
  related_code: State
}

const RelatedCodeHeader = (props: RelatedCodeHeaderProps) => {
  const { related_code: { location, status, relative_path, filename, project_tag }} = props
  if (status !== Status.NoSearch) {

    let line = location && location.line ? `LINE ${location.line}` : ""

    return (
      <div className={styles.container}>
        <div className={styles.title_container}>
          <div className={styles.graphic}/>
          <div className={styles.title}>
            <div className={styles.filepath}>
              {relative_path}
            </div>
            <div className={styles.filename}>
              {filename}
            </div>
            <div className={styles.file_line}>
              {line}
            </div>
          </div>
        </div>
        <div className={styles.subtitle}>
          <div className={styles.subtitle_label}>
            RELATED CODE
            <div
              className={styles.subtitle_tooltip}
              onClick={() => electron.shell.openExternal(`https://${Domains.Help}/article/147-find-related-code-in-the-copilot`)}
            >
              ?
            </div>
          </div>
          <div className={styles.project_tag}>
            <div className={styles.icon_git} />
            {project_tag}
          </div>
        </div>
      </div>
    )
  }
  return (
    <div className={styles.container}>
      <div className={styles.title_container}>
        <div className={styles.graphic}/>
        <div className={styles.title}>
          <h4>Related Code Finder</h4>
        </div>
      </div>
    </div>
  )
}

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    related_code: state.related_code,
  }
}

export default connect(mapStateToProps, null)(RelatedCodeHeader)
