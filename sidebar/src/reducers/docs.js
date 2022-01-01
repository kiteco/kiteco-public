import * as actions from '../actions/docs'

const defaultState = {
  status: "loading",
  data: {},
  language: null,
  identifier: null,
  error: null,
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

const integrateMembers = (state, members) => {
  const kind = state.data.value.kind
  return Object.assign({}, state, {
    data: Object.assign({}, state.data, {
      value: Object.assign({}, state.data.value, {
        details: Object.assign({}, state.data.value.details, {
          [kind]: Object.assign({}, state.data.value.details[kind], {
            members: members.members
          })
        })
      })
    })
  })
}

const showMembers = (state, action) => {
  return {
    ...integrateMembers(state, action.members),
    status: "success",
  }
}

const loadMembersFailed = (state, action) => {
  return {
    ...state,
    status: "failed",
  }
}

// const logOut = () => ({ ...defaultState })

const docs = (state = defaultState, action) => {
  switch (action.type) {
    case actions.LOAD_DOCS:
      return loadDocs(state, action);
    case actions.SHOW_DOCS:
      return showDocs(state, action);
    case actions.LOAD_DOCS_FAILED:
      return loadDocsFailed(state, action);
    case actions.LOAD_MEMBERS:
      return loadMembers(state, action);
    case actions.SHOW_MEMBERS:
      return showMembers(state, action);
    case actions.LOAD_MEMBERS_FAILED:
      return loadMembersFailed(state, action);
    // case actions.LOG_OUT:
    //   return logOut();
    default:
      return state;
  }
}

export default docs
