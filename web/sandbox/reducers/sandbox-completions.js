import * as actions from '../actions/sandbox-completions'

const RESPONSE_HISTORY_SIZE = 15
const RESPONSE_TIME_THRESHOLD = 0.5
const RESPONSE_TIME_SLOW = 200 //ms

const defaultState = {
  completionsMap: {},
  fileTooLargeMap: {},
  responseTimes: null,
  currentResponseTimeIdx: 0,
  currentOverLatencyCount: 0,
  completionsHaveLatency: false,
  skipCompletionsMap: {},
}

const registerCompletions = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    completionsMap: {
      ...state.completionsMap,
      [key]: action.completions
    },
    fileTooLargeMap: {
      ...state.fileTooLargeMap,
      [key]: false
    },
    skipCompletionsMap: {
      ...state.skipCompletionsMap,
      [key]: false,
    }
  }
}

const loadCompletionsFailed = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    completionsMap: {
      ...state.completionsMap,
      [key]: []
    }
  }
}

const completionsFileTooLarge = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    completionsMap: {
      ...state.completionsMap,
      [key]: []
    },
    fileTooLargeMap: {
      ...state.fileTooLargeMap,
      [key]: true,
    }
  }
}

const completionsFileCleared = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    completionsMap: {
      ...state.completionsMap,
      [key]: []
    },
    fileTooLargeMap: {
      ...state.fileTooLargeMap,
      [key]: false,
    }
  }
}

const clearCompletions = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    completionsMap: {
      ...state.completionsMap,
      [key]: []
    }
  }
}

const logCompletionsResponseTime = (state, action) => { //action.times
  let responseTimes = new Array(RESPONSE_HISTORY_SIZE)
  const { start=0, end=0 } = action.times
  if(state.responseTimes === null) {
    responseTimes.fill(0)
  } else {
    responseTimes = state.responseTimes.slice()
  }

  let currentResponseTimeIdx = state.currentResponseTimeIdx % RESPONSE_HISTORY_SIZE

  const diff = end - start
  const prevDiff = responseTimes[currentResponseTimeIdx]
  responseTimes[currentResponseTimeIdx] = diff

  let currentOverLatencyCount = state.currentOverLatencyCount

  if(diff > RESPONSE_TIME_SLOW && prevDiff <= RESPONSE_TIME_SLOW) {
    currentOverLatencyCount++
  } else if (diff <= RESPONSE_TIME_SLOW && prevDiff > RESPONSE_TIME_SLOW) {
    currentOverLatencyCount--
  }

  let completionsHaveLatency = false
  if(currentOverLatencyCount / RESPONSE_HISTORY_SIZE > RESPONSE_TIME_THRESHOLD) {
    completionsHaveLatency = true
  }

  currentResponseTimeIdx += 1

  return {
    ...state,
    responseTimes,
    currentResponseTimeIdx,
    completionsHaveLatency,
    currentOverLatencyCount,
  }
}

const skipCompletions = (state, action) => {
  const key = action.editorId+action.filename
  return {
    ...state,
    skipCompletionsMap: {
      ...state.skipCompletionsMap,
      [key]: action.skip,
    }
  }
}

const sandboxCompletions = (state = defaultState, action) => {
  switch(action.type) {
    case actions.LOAD_COMPLETIONS_FAILED:
      return loadCompletionsFailed(state, action)
    case actions.COMPLETIONS_FILE_TOO_LARGE:
      return completionsFileTooLarge(state, action)
    case actions.REGISTER_COMPLETIONS:
      return registerCompletions(state, action)
    case actions.CLEAR_COMPLETIONS:
      return clearCompletions(state, action)
    case actions.COMPLETIONS_FILE_CLEARED:
      return completionsFileCleared(state, action)
    case actions.SKIP_COMPLETIONS:
      return skipCompletions(state, action)
    case actions.LOG_COMPLETIONS_RESPONSE_TIME:
      const retVal = logCompletionsResponseTime(state, action)
      return retVal   
    default:
      return state
  }
}

export default sandboxCompletions