import { Action as BaseAction } from 'redux'
import { ThunkAction, ThunkDispatch } from 'redux-thunk'
import { POST } from "../../actions/fetch"
import { FetchRequest, FetchResponse, Location, OpenFileRequest, Push, RelatedFile } from "./api-types"

const { ipcRenderer } = window.require('electron')

enum ActionType {
  Error = "relatedcode.error",
  LoadResults = "relatedcode.loadresults",
  NewSearch = "relatedcode.newsearch",
  Stale = "relatedcode.stale",
}

interface ErrorAction extends BaseAction {
  type: ActionType.Error
  data: string
}

interface LoadResultsAction extends BaseAction {
  type: ActionType.LoadResults
  data: FetchResponse
}

interface NewSearchAction extends BaseAction {
  type: ActionType.NewSearch
  data: Push
}

interface StaleAction extends BaseAction {
  type: ActionType.Stale
}

export enum Status {
  NoSearch = "NoSearch",
  Loading = "Loading",
  Loaded = "Loaded",
  NoMoreResults = "NoMoreResults",
  Stale = "Stale",
  Error = "Error"
}

export interface State {
  status: Status,

  editor: string,
  editor_install_path: string,
  error: string,
  project_tag: string,
  project_root: string,
  location?: Location,
  relative_filename: string,
  relative_path: string,
  filename: string,
  related_files: RelatedFile[],
}

const defaultState: State = {
  status: Status.NoSearch,

  editor: "",
  editor_install_path: "",
  error: "",
  project_root: "",
  project_tag: "",
  location: undefined,
  relative_filename: "",
  relative_path: "",
  filename: "",
  related_files: [],
}

const LoadResultsParams = {
  num_files: 20,
  num_blocks: 3,
  num_keywords: 3,
}

export function loadMoreResults(state: State): ThunkAction<Promise<void>, any, {}, Action> {
  const { location, related_files, status } = state
  if (status === Status.NoSearch || !location) {
    console.error("unable to load more results: bad search state")
    return () => Promise.resolve()
  }
  return async (dispatch): Promise<void> => {
    return requestRelated(location, related_files.length, state.editor, dispatch)
  }
}

export function openFile(
  state: State,
  appPath: string,
  filename: string,
  line: number,
  file_rank: number,
  block_rank?: number,
): ThunkAction<Promise<void>, any, {}, Action> {
  return (dispatch) => {
    const request: OpenFileRequest = {
      path: appPath,
      filename: filename,
      line,
      block_rank,
      file_rank,
    }
    return dispatch(POST({
      url: "/clientapi/plugins/" + state.editor + "/open",
      options: {
        body: JSON.stringify(request),
      },
    })).then(({ success, error }: any) => {
      if (!success && error) {
        throw new Error(error)
      }
    })
  }
}

export function reloadSearch(state: State): ThunkAction<void, {}, {}, Action> {
  const { location, status } = state
  if (status === Status.NoSearch || !location) {
    console.error("unable to reload search: bad search state")
    return () => Promise.resolve()
  }
  return relatedCodeEvent({
    editor: state.editor,
    editor_install_path: state.editor_install_path,
    location: state.location,
    relative_filename: state.relative_filename,
    filename: state.filename,
    relative_path: state.relative_path,
    project_tag: state.project_tag,
  } as Push)
}

function requestRelated(location: Location, offset: number, editor: string, dispatch: ThunkDispatch<any, {}, Action>): Promise<void> {
  let request: FetchRequest = {
    ...LoadResultsParams,
    location,
    offset,
    editor,
  }
  return dispatch(POST({
    url: "/codenav/related",
    options: {
      body: JSON.stringify(request),
    },
  }))
    .then(({ success, data, status, error }: any) => {
      if (success) {
        dispatch({
          type: ActionType.LoadResults,
          data: JSON.parse(data),
        })
      } else if (status === 405) {
        dispatch({
          type: ActionType.Stale,
        })
      } else {
        dispatch({
          type: ActionType.Error,
          data: error,
        })
      }
    })
}

export function relatedCodeEvent(push: Push): ThunkAction<void, {}, {}, Action> {
  return (dispatch) => {
    // Push the loading state
    dispatch({
      type: ActionType.NewSearch,
      data: push,
    })
    // Fetch initial results and update state
    requestRelated(push.location, 0, push.editor, dispatch)
  }
}

type Action = ErrorAction | LoadResultsAction | NewSearchAction | StaleAction

export function reducer(state: State = defaultState, action: Action): any {
  switch (action.type) {
    case ActionType.Error:
      console.error("related code error:", action.data)
      return {
        ...state,
        error: action.data,
        status: Status.Error,
      }
    case ActionType.LoadResults:
      return {
        ...state,
        status: action.data.related_files.length < LoadResultsParams.num_files ? Status.NoMoreResults : Status.Loaded,
        project_root: action.data.project_root,
        related_files: state.related_files.concat(action.data.related_files),
      }
    case ActionType.NewSearch:
      ipcRenderer.send('focus-window')
      return {
        ...defaultState,
        id: Date.now(),
        status: Status.Loading,
        editor: action.data.editor,
        editor_install_path: action.data.editor_install_path,
        location: action.data.location,
        project_tag: action.data.project_tag,
        filename: action.data.filename,
        relative_path: action.data.relative_path,
      }
    case ActionType.Stale:
      return {
        ...state,
        status: Status.Stale,
      }
    default:
      return state
  }
}