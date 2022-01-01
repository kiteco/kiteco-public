import * as actions from '../actions/docs'
import { LOG_OUT } from '../actions/account'
import { integrateMembers } from '../../utils/data-normalization'

const defaultState = {
  status: "loading",
  memberSortCriteria: "popularity",
  data: {},
  language: null,
  identifier: null,
  exampleId: null,
  kind: null,
  error: null,
}

const setExample = (state, action) => {
  return {
    ...state,
    language: action.language,
    exampleId: action.exampleId,
    status: "loading",
  }
}

const setPageKind = (state, action) => {
  return {
    ...state,
    kind: action.kind,
  }
}

const showDocs = (state, action) => {
  return {
    ...state,
    status: "success",
    data: action.data,
  }
}

const loadDocsFailed = (state, action) => {
  return {
    ...state,
    status: "failed",
    error: action.error,
  }
}

const loadDocs = (state, action) => {
  return {
    ...state,
    status: "loading",
    language: action.language,
    identifier: action.identifier,
  }
}

const loadMembers = (state, action) => {
  return {
    ...state,
    status: "loading",
    language: action.language,
    identifier: action.identifier,
  }
}

const memCmp = (field, isAsc) => (memA, memB) => isAsc 
  ? memA[field] > memB[field]
  : memA[field] < memB[field]

const sortMembers = (state, action) => {
  let members;
  switch(action.criteria) {
    case 'name':
      members = state.data.members.slice(0).sort(memCmp('name', true))
      break
    default:
      members = state.data.members.slice(0).sort(memCmp('popularity', false))
  }
  return {
    ...state,
    memberSortCriteria: action.criteria,
    data: {
      ...state.data,
      members
    }
  }
}

const showMembers = (state, action) => {
  return {
    ...state,
    data: integrateMembers(state.data, action.members, state.language),
    status: "success",
  }
}

const loadMembersFailed = (state, action) => {
  return {
    ...state,
    error: action.error,
    status: "failed",
  }
}

const logOut = () => ({ ...defaultState })

const docs = (state = defaultState, action) => {
  switch (action.type) {
    case actions.SET_EXAMPLE:
      return setExample(state, action)
    case actions.LOAD_DOCS:
      return loadDocs(state, action);
    case actions.SET_PAGE_KIND:
      return setPageKind(state, action)
    case actions.SHOW_DOCS:
      return showDocs(state, action);
    case actions.LOAD_DOCS_FAILED:
      return loadDocsFailed(state, action);
    case actions.LOAD_MEMBERS:
      return loadMembers(state, action)
    case actions.SHOW_MEMBERS:
      return showMembers(state, action)
    case actions.LOAD_MEMBERS_FAILED:
      return loadMembersFailed(state, action)
    case actions.SORT_MEMBERS:
      return sortMembers(state, action)
    case LOG_OUT:
      return logOut();
    default:
      return state;
  }
}

export default docs
