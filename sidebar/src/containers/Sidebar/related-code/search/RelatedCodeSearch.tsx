import * as React from "react"
import { connect } from "react-redux"
import { ThunkDispatch } from "redux-thunk"
import { AnyAction } from "redux"

import { loadMoreResults, reloadSearch, State, Status } from '../../../../store/related-code/related-code'
import { RelatedFile } from "../../../../store/related-code/api-types"

import RelatedCodeResult from "./RelatedCodeResult"
import Spinner from "../../../../components/Spinner"

import styles from './related-code-search.module.css'

interface RelatedCodeSearchProps {
  loadMoreResults: (state: State) => Promise<void>,
  reloadSearch: (state: State) => void,
  related_code: State
}

/*
	RelatedCodeSearch displays an active Related Code search
 */
const RelatedCodeSearch = (props: RelatedCodeSearchProps) =>  {
  if (props.related_code.status === Status.Loading) {
    return (
      <div className={styles.container}>
        <Spinner theme="dark" text="Searching for related code..."/>
      </div>
    )
  }
  if (props.related_code.status === Status.NoMoreResults && props.related_code.related_files.length === 0) {
    return (
      <div className={styles.container}>
        <div className={styles.no_results}>
          <div className={styles.no_results_text}>
            Kite could not find any files related to <span>{props.related_code.filename}</span>
          </div>
          <div className={styles.warning_icon}/>
        </div>
      </div>
    )
  }
  if (props.related_code.status === Status.Error) {
    return (
      <div className={styles.container}>
        <div className={styles.no_results}>
          <div className={styles.error_text}>
            An error has occurred:
            <br />
            {props.related_code.error}
          </div>
          <div className={styles.warning_icon}/>
        </div>
      </div>
    )
  }
  return (
    <div className={styles.container}>
      <div className={styles.results_list}>
        {
          props.related_code.related_files.map((result: RelatedFile, index: number) =>
            <RelatedCodeResult
              file_rank={index+1}
              key={result.relative_path + result.filename}
              result={result}
            />
          )
        }
      </div>
      <ActionBar {...props} />
    </div>
  )
}

const ActionBar = (props: RelatedCodeSearchProps) => {
  switch (props.related_code.status) {
    case Status.NoMoreResults:
      return (
        <div className={styles.action_bar}>
          No More Results
        </div>
      )
    case Status.Stale:
      return (
        <div
          className={`${styles.action_bar} ${styles.action}`}
          onClick={() => props.reloadSearch(props.related_code)}
        >
          <div className={styles.load_icon} />
          Search is out of date. Click here to reload.
        </div>
      )
    default:
      return (
        <div
          className={`${styles.action_bar} ${styles.action}`}
          onClick={() => props.loadMoreResults(props.related_code)}
        >
          <div className={styles.load_icon} />
          Load More
        </div>
      )
  }
}

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  loadMoreResults: (state: State) => dispatch(loadMoreResults(state)),
  reloadSearch: (state: State) => dispatch(reloadSearch(state)),
})

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    related_code: state.related_code,
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(RelatedCodeSearch)