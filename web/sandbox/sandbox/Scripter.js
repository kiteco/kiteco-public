import React from 'react'
import { connect } from 'react-redux'

import difference from 'lodash/difference'

import { arrayEquals } from '../utils/functional'

import { skipCompletions } from '../actions/sandbox-completions'

import { 
  typeKey, 
  setCompletionIdx, 
  incrementCompletionIdx, 
  selectCompletion,
  setBuffer,
  setCursor,
  setCursorBlink,
  concatToBuffer,
  hideCaption,
  showCaption,
  resetCompletionUI,
  showOverlay,
  showCursorCaption,
  hideCursorCaption,
} from '../actions/sandbox-editors'
import { 
  TYPE_CHARS, 
  TYPE_COMPLETION, 
  TYPE_HELPER, 
  TYPE_PAUSE,
  DEFAULT_LATENCY_CHAR,
  DEFAULT_LATENCY_PAUSE,
  DEFAULT_LATENCY_COMPLETION,
  DEFAULT_LATENCY_INTER_ACTION,
  FALLBACK_LATENCY_CHAR,
  AMOUNT,
  scripts,
  DEFAULT_PRE_COMPLETION_LATENCY,
  TIMEOUT_BUFFER_RATIO,
  TYPE_CAPTION,
  DEFAULT_LATENCY_FINAL_SELECTION,
  RIGHT_CURSOR_CAPTION_PLACEMENT,
} from './scripts'

const NOT_READY = 'not ready'
const RESTARTING = 'restarting'
const AUTOFILL = 'autofill'
const PAUSE = 'pause'
const RESET = 'reset'
const EDIT = 'edit'
const UNKNOWN = 'unknown'

const DEFAULT_INSTANCE_VALS = {
  autofillScript: false,
  readyToScript: false,
  restartScript: false,
  pauseScript: false,
  scriptReset: false,
  scriptEdit: false,
  scriptCompleted: false,
  nextScriptingFn: null,
  shortCircuitActionFn: null,
  currentPause: null,
  resumptionCaption: null,
}

const SCRIPT_RESUMPTION_DELAY = 500 //ms

class Scripter extends React.Component {

  constructor(props) {
    super(props)
    this.scripterMap = {}
    this.props.filenames.forEach(this.initializeMapForName)
  }

  initializeMapForName = name => {
    this.scripterMap[name] = { ...DEFAULT_INSTANCE_VALS }
  }

  componentDidUpdate(prevProps) {
    if(!arrayEquals(prevProps.filenames, this.props.filenames)) {
      const oldFiles = difference(prevProps.filenames, this.props.filenames)
      oldFiles.forEach(name => {
        delete this.scripterMap[name]
      })
      const newFiles = difference(this.props.filenames, prevProps.filenames)
      newFiles.forEach(this.initializeMapForName)
    }
    if(prevProps.activeScriptName !== this.props.activeScriptName) {
      this.pauseScripting(prevProps.activeScriptName)
      this.scriptingReady(this.props.activeScriptName)
      // debouncing-y technique to prevent weird behavior on rapid script switches
      clearTimeout(this.resumeScriptingTimeout)
      this.resumeScriptingTimeout = setTimeout(() => {
        this.resumeScripting(this.props.activeScriptName)
      }, SCRIPT_RESUMPTION_DELAY)
    }
  }

  shouldReject(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return !this.scripterMap[scriptName].readyToScript 
      || this.scripterMap[scriptName].restartScript 
      || this.scripterMap[scriptName].autofillScript 
      || this.scripterMap[scriptName].pauseScript 
      || this.scripterMap[scriptName].scriptReset
      || this.scripterMap[scriptName].scriptEdit
  }

  autoplayFinished(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].scriptReset
      || this.scripterMap[scriptName].scriptEdit
      || this.scripterMap[scriptName].scriptCompleted
  }

  activelyScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return !this.scripterMap[scriptName].pauseScript 
      && !this.scripterMap[scriptName].scriptReset 
      && !this.scripterMap[scriptName].scriptEdit
      && !this.scripterMap[scriptName].scriptCompleted
  }

  rejectionReason(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.scripterMap[scriptName].readyToScript) return NOT_READY
    if(this.scripterMap[scriptName].restartScript) return RESTARTING
    if(this.scripterMap[scriptName].autofillScript) return AUTOFILL
    if(this.scripterMap[scriptName].pauseScript) return PAUSE
    if(this.scripterMap[scriptName].scriptReset) return RESET
    if(this.scripterMap[scriptName].scriptEdit) return EDIT
    return UNKNOWN
  }

  scriptingReady(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].readyToScript = true
  }

  scriptPause(pauseAmount, scriptIter, inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.shouldReject(scriptName)) {
      const pauseTyping = new Promise((resolve, reject) => {
        const actionHandler = () => {
          this.scripterMap[scriptName].shortCircuitActionFn = null
          resolve()
        }
        this.scripterMap[scriptName].currentPause = pauseAmount
        const handle = setTimeout(() => {
          actionHandler()
        }, pauseAmount)
        this.scripterMap[scriptName].shortCircuitActionFn = () => {
          clearTimeout(handle)
          actionHandler()
        }
      })
      return pauseTyping.then(() => {
        this.scripterMap[scriptName].currentPause = null
        return this.scriptingHelper(scriptIter, scriptName)
      })
    } else {
      return Promise.reject({
        reason: this.rejectionReason(scriptName),
        source: TYPE_PAUSE,
        scriptIter,
      })
    }
  }

  scriptCaption(captionObj, scriptIter, inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.shouldReject(scriptName)) {
      const captionAction = new Promise((resolve, reject) => {
        if(captionObj.hide) {
          this.props.hideCaption(this.props.editorId)
          resolve(false)
        } else {
          this.props.showCaption(this.props.editorId, captionObj.caption, captionObj.completionCaptionPadding)
          resolve(true)
        }
      })
      return captionAction.then(showing => {
        if(showing) {
          //set resume flag appropriately
          this.scripterMap[scriptName].resumptionCaption = captionObj
        } else {
          //unset
          this.scripterMap[scriptName].resumptionCaption = null
        }
        return this.scriptingHelper(scriptIter, scriptName)
      })
    } else {
      return Promise.reject({
        reason: this.rejectionReason(scriptName),
        source: TYPE_CAPTION,
        captionObj,
        scriptIter,
      })
    }
  }

  scriptChars(charIter, pauseAmount, skipCompletions, scriptIter, inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.shouldReject(scriptName)) {
      const charTyping = new Promise((resolve, reject) => {
        const {value, done} = charIter.next()

        const actionHandler = () => {
          if(!done) {
            // gated to prevent timeout from doing laggy typing
            if(!this.autoplayFinished(scriptName)) {
              this.props.typeKey(value, this.props.editorId, scriptName)
            }
            resolve(false)
          } else {
            resolve(true)
          }
          this.scripterMap[scriptName].shortCircuitActionFn = null
        }
        this.scripterMap[scriptName].currentPause = pauseAmount
        const handle = setTimeout(() => {
          actionHandler()
        }, pauseAmount)

        this.scripterMap[scriptName].shortCircuitActionFn = () => {
          clearTimeout(handle)
          actionHandler()
        }
      })
      return charTyping.then((done) => {
        this.scripterMap[scriptName].currentPause = null
        if(!done) {
          return this.scriptChars(charIter, pauseAmount, skipCompletions, scriptIter, scriptName)
        } else {
          if(skipCompletions) {
            this.props.skipCompletions(false, this.props.editorId, scriptName)
          }
          return this.scriptingHelper(scriptIter, scriptName)
        }
      })
    } else {
      return Promise.reject({
        reason: this.rejectionReason(scriptName),
        source: TYPE_CHARS,
        charIter,
        pauseAmount,
        skipCompletions,
        scriptIter
      })
    }
  }

  scriptCompletion(numMoves,
     pauseAmount,
     finalPause,
     completionAutofill,
     cursorCaption,
     cursorCaptionPlacement,
     captionMarginTop,
     captionMarginLeft,
     captionAfterClass,
     scriptIter,
     inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.shouldReject(scriptName)) {
      const completionSelecting = new Promise((resolve, reject) => {
        if(cursorCaption && !this.props.cursorCaption) {
          //it takes 5 moves to move the highlight the bottom of the completions box
          this.props.showCursorCaption(this.props.editorId, cursorCaption, cursorCaptionPlacement, captionMarginTop, captionMarginLeft, captionAfterClass, Math.min(numMoves, 5))
        }
        const actionHandler = () => {
          if(this.isPaused(scriptName) && scriptName !== this.props.activeScriptName) {
            // need the completion value passed in
            this.props.concatToBuffer(completionAutofill, this.props.editorId, scriptName)
            resolve({ done: true, doNotComplete: true })
          } else if(numMoves === -1) {
            // then did not find intended completion
            // need error case handling
            resolve({ couldNotFind: true, completionAutofill })
          } else if(numMoves > 0) {
            resolve({ done: false, completionAutofill })
          } else {
            resolve({ done: true })
          }
          this.scripterMap[scriptName].shortCircuitActionFn = null
        }

        if(numMoves === 0) {
          pauseAmount += finalPause
        }

        const handle = setTimeout(() => {
          actionHandler()
        }, pauseAmount)

        this.scripterMap[scriptName].shortCircuitActionFn = () => {
          clearTimeout(handle)
          this.props.hideCursorCaption(this.props.editorId)
          this.props.concatToBuffer(completionAutofill, this.props.editorId, scriptName)
          resolve({ done: true, doNotComplete: true })
        }
      })

      return completionSelecting.then(({ done, couldNotFind, completionAutofill, doNotComplete=false }) => {
        // only perform actions if this is for the activeScript
        if(couldNotFind) {
          this.props.hideCursorCaption(this.props.editorId)
          this.props.resetCompletionUI(this.props.editorId)
          this.props.skipCompletions(true, this.props.editorId, scriptName)
          return this.scriptChars(completionAutofill[Symbol.iterator](), FALLBACK_LATENCY_CHAR, true, scriptIter, scriptName)
        } else if(!done) {
          this.props.incrementCompletionIdx(this.props.editorId)
          return this.scriptCompletion(numMoves-1,
                                      pauseAmount,
                                      finalPause,
                                      completionAutofill,
                                      cursorCaption,
                                      cursorCaptionPlacement,
                                      captionMarginTop,
                                      captionMarginLeft,
                                      captionAfterClass,
                                      scriptIter,
                                      scriptName)
        } else {
          this.props.hideCursorCaption(this.props.editorId)
          if(!doNotComplete) {
            this.props.selectCompletion(this.props.editorId)
          }
          return this.scriptingHelper(scriptIter, scriptName)
        }
      })
    } else {
      return Promise.reject({
        reason: this.rejectionReason(scriptName),
        source: TYPE_COMPLETION,
        numMoves,
        pauseAmount,
        completionAutofill,
        cursorCaption,
        cursorCaptionPlacement,
        captionMarginTop,
        captionMarginLeft,
        captionAfterClass,
        scriptIter
      })
    }
  }

  //TODO(dane) should this be made cancellable??
  addPause(amount) {
    const pausePromise = new Promise((resolve, reject) => {
      setTimeout(() => {
        resolve()
      }, amount)
    })
    return pausePromise
  }

  // may have to clean up a bit
  scriptingHelper(scriptIter, inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(!this.shouldReject(scriptName)) {
      const scriptPiece = new Promise((resolve, reject) => {
        const {value, done} = scriptIter.next()
        const actionHandler = () => {
          if(done) {
            resolve({ done })
          } else {
            resolve({ done, value })
          }
          this.scripterMap[scriptName].shortCircuitActionFn = null
        }

        this.scripterMap[scriptName].currentPause = DEFAULT_LATENCY_INTER_ACTION
        const handle = setTimeout(() => {
          actionHandler()
        }, DEFAULT_LATENCY_INTER_ACTION)

        this.scripterMap[scriptName].shortCircuitActionFn = () => {
          clearTimeout(handle)
          actionHandler()
        }
      })
      return scriptPiece.then(({ done, value }) => {
        this.scripterMap[scriptName].currentPause = null
        if(!done) {
          switch(value.type) {
            case TYPE_CHARS:
              const charPause = value.hasOwnProperty(AMOUNT)
                ? value[AMOUNT]
                : DEFAULT_LATENCY_CHAR
              const skipCompletions = value.skipCompletions || false
              if(skipCompletions) {
                this.props.skipCompletions(skipCompletions, this.props.editorId, scriptName)
              }
              return this.scriptChars(value.sequence[Symbol.iterator](), charPause, skipCompletions, scriptIter, scriptName)
            case TYPE_COMPLETION:
              return this.addPause(DEFAULT_PRE_COMPLETION_LATENCY)
                .then(() => {
                  // assume zero index start
                  const key = this.props.editorId+scriptName
                  let numMoves = -1
                  if(this.props.completionsMap[key]) {
                      numMoves = this.props.completionsMap[key].findIndex(completion => {
                      return completion.display === value.select
                    })
                  }
                  // in case remote fetch has failed
                  if(numMoves === -1 
                  && this.props.cachedCompletions
                  && this.props.cachedCompletions[this.props.cursorPos]) {
                    numMoves = this.props.cachedCompletions[this.props.cursorPos].findIndex(completion => {
                      return completion.display === value.select
                    })
                  }
                  this.props.setCompletionIdx(0, this.props.editorId)
                  const completionPause = value.hasOwnProperty(AMOUNT)
                    ? value[AMOUNT]
                    : DEFAULT_LATENCY_COMPLETION
                  const completionAutofill = value.complete || value.select
                  const finalPause = value.finalSelectionWait || DEFAULT_LATENCY_FINAL_SELECTION
                  const captionPlacement = value.cursorCaptionPlacement || RIGHT_CURSOR_CAPTION_PLACEMENT
                  const marginTop = value.marginTop || 0
                  const marginLeft = value.marginLeft || 0
                  const afterClass = value.afterClass || ''
                  return this.scriptCompletion(numMoves, 
                                              completionPause, 
                                              finalPause, 
                                              completionAutofill, 
                                              value.cursorCaption, 
                                              captionPlacement,
                                              marginTop,
                                              marginLeft,
                                              afterClass, 
                                              scriptIter, 
                                              scriptName)
                })
            case TYPE_PAUSE:
              const pausePause = value.hasOwnProperty(AMOUNT)
                ? value[AMOUNT]
                : DEFAULT_LATENCY_PAUSE
              return this.scriptPause(pausePause, scriptIter, scriptName)
            case TYPE_CAPTION:
                return this.scriptCaption(value, scriptIter, scriptName)
            default:
              break
          }
        } else {
          return Promise.resolve()
        }
      })
    } else {
      return Promise.reject({
        reason: this.rejectionReason(scriptName),
        source: TYPE_HELPER,
        scriptIter
      })
    }
  }

  resumeScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    // only resume if ready and not editing
    if(this.scripterMap[scriptName].readyToScript && !this.scripterMap[scriptName].scriptEdit) {
      this.scripterMap[scriptName].pauseScript = false
      if(this.scripterMap[scriptName].resumptionCaption) {
        this.props.showCaption(this.props.editorId, 
          this.scripterMap[scriptName].resumptionCaption.caption, 
          this.scripterMap[scriptName].resumptionCaption.completionCaptionPadding)
          this.scripterMap[scriptName].resumptionCaption = null
      }
      if(this.scripterMap[scriptName].nextScriptingFn) {
        this.scripterMap[scriptName].nextScriptingFn()
            .then(this.scriptLooper.bind(this, scriptName))
            .catch(this.reasonCatcher.bind(this, scriptName))
        this.scripterMap[scriptName].nextScriptingFn = null
      } else {
        this.startScripting(scriptName)
      }
    }
  }

  startScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(this.scripterMap[scriptName].readyToScript) {
      this.scripterMap[scriptName].nextScriptingFn = null
      this.scripterMap[scriptName].scriptReset = false
      this.scripterMap[scriptName].scriptEdit = false
      this.scripterMap[scriptName].pauseScript = false
      this.scripterMap[scriptName].autofillScript = false
      this.scripterMap[scriptName].scriptCompleted = false
      this.scripterMap[scriptName].restartScript = false
      this.scripterMap[scriptName].resumptionCaption = null
      this.props.hideCaption(this.props.editorId)
      const script = scripts[scriptName]
      this.props.setBuffer(script.startBuffer, this.props.editorId, scriptName)
      if(script.initialCursor) {
        this.props.setCursor(script.initialCursor, this.props.editorId)
      }
      this.scriptingHelper(script.script[Symbol.iterator](), scriptName)
        .then(this.scriptLooper.bind(this, scriptName))
        .catch(this.reasonCatcher.bind(this, scriptName))
    }
  }

  scriptLooper(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(this.props.loop) {
      this.props.setBuffer(scripts[scriptName].startBuffer, this.props.editorId, scriptName)
      this.startScripting(scriptName)
    } else {
      this.scripterMap[scriptName].scriptCompleted = true
      // show overlay on completion of scripting
      this.props.showOverlay(this.props.editorId)
    }
  }

  reasonCatcher(inputName, { reason, source, ...scriptState }) {
    const scriptName = inputName || this.props.activeScriptName
    // let's set this to the target buffer state
    const script = scripts[scriptName]
    switch(reason) {
      case AUTOFILL:
        this.props.setBuffer(script.filledBuffer, this.props.editorId, scriptName)
        break
      case RESTARTING:
        this.props.setBuffer(script.startBuffer, this.props.editorId, scriptName)
        this.scripterMap[scriptName].restartScript = false
        this.startScripting(scriptName)
        break
      case RESET:
        this.props.setBuffer(script.startBuffer, this.props.editorId, scriptName)
        break
      case EDIT:
        break
      case PAUSE:
        switch(source) {
          case TYPE_CHARS:
            this.scripterMap[scriptName].nextScriptingFn = this.scriptChars.bind(
                                                                                  this, 
                                                                                  scriptState.charIter, 
                                                                                  scriptState.pauseAmount,
                                                                                  scriptState.skipCompletions, 
                                                                                  scriptState.scriptIter, 
                                                                                  scriptName
                                                                                )
            break
          case TYPE_COMPLETION:
            this.scripterMap[scriptName].nextScriptingFn = this.scriptCompletion.bind(
                                                                                      this, 
                                                                                      scriptState.numMoves, 
                                                                                      scriptState.pauseAmount, 
                                                                                      scriptState.completionAutofill,
                                                                                      scriptState.cursorCaption, 
                                                                                      scriptState.cursorCaptionPlacement,
                                                                                      scriptState.captionMarginTop,
                                                                                      scriptState.captionMarginLeft,
                                                                                      scriptState.captionAfterClass,
                                                                                      scriptState.scriptIter, 
                                                                                      scriptName
                                                                                    )
            break
          case TYPE_CAPTION:
            this.scripterMap[scriptName].nextScriptingFn = this.scriptCaption.bind(
                                                                                    this,
                                                                                    scriptState.captionObj,
                                                                                    scriptState.scriptIter,
                                                                                    scriptName                                                                      
                                                                                  )
            break    
          case TYPE_PAUSE:
          case TYPE_HELPER:
            this.scripterMap[scriptName].nextScriptingFn = this.scriptingHelper.bind(this, scriptState.scriptIter, scriptName)
            break
          default:
            //autofill here
            this.scripterMap[scriptName].pauseScript = false
            this.scripterMap[scriptName].nextScriptingFn = null
            this.props.setBuffer(script.filledBuffer, this.props.editorId, scriptName)
            break
        }
        break
      default:
        break
    }
  }

  scriptingDone(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].nextScriptingFn = null
    this.scripterMap[scriptName].readyToScript = false
  }

  finishScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].nextScriptingFn = null
    this.scripterMap[scriptName].autofillScript = true
    this.scripterMap[scriptName].scriptEdit = true
    this.scripterMap[scriptName].shortCircuitActionFn && this.scripterMap[scriptName].shortCircuitActionFn()
    this.scripterMap[scriptName].shortCircuitActionFn = null
  }

  restartScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].restartScript = true
    this.scripterMap[scriptName].shortCircuitActionFn && this.scripterMap[scriptName].shortCircuitActionFn()
    this.scripterMap[scriptName].shortCircuitActionFn = null
    this.scripterMap[scriptName].nextScriptingFn = null
    if(this.isAutofilled(scriptName) || this.isEditing(scriptName)) {
      this.startScripting(scriptName)
    }
  }

  pauseScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    if(this.scripterMap[scriptName].readyToScript) {
      this.scripterMap[scriptName].shortCircuitActionFn && this.scripterMap[scriptName].shortCircuitActionFn()
      this.scripterMap[scriptName].shortCircuitActionFn = null
      this.scripterMap[scriptName].pauseScript = true
    }
  }

  editScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].pauseScript = false
    this.scripterMap[scriptName].scriptEdit = true
    this.scripterMap[scriptName].nextScriptingFn = null
    this.scripterMap[scriptName].shortCircuitActionFn && this.scripterMap[scriptName].shortCircuitActionFn()
    this.scripterMap[scriptName].shortCircuitActionFn = null
  }

  resetScripting(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].autofillScript = false
    this.scripterMap[scriptName].restartScript = false
    this.scripterMap[scriptName].pauseScript = false
    this.scripterMap[scriptName].scriptCompleted = false
    this.scripterMap[scriptName].scriptReset = true
    this.scripterMap[scriptName].nextScriptingFn = null
    this.scripterMap[scriptName].shortCircuitActionFn && this.scripterMap[scriptName].shortCircuitActionFn()
    this.scripterMap[scriptName].shortCircuitActionFn = null
    if(!this.activelyScripting(scriptName)) {
      this.props.setBuffer(scripts[scriptName].startBuffer, this.props.editorId, scriptName)
    }
  }

  invalidatePauseState(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    this.scripterMap[scriptName].pauseScript = false
    this.scripterMap[scriptName].scriptEdit = true
    this.scripterMap[scriptName].nextScriptingFn = null
  }

  isPaused(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].pauseScript
  }

  isReset(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].scriptReset
  }

  isAutofilled(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].autofillScript
  }

  isEditing(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].scriptEdit
  }

  isScriptCompleted(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].scriptCompleted
  }

  currentPause(inputName) {
    const scriptName = inputName || this.props.activeScriptName
    return this.scripterMap[scriptName].currentPause
  }
  
  render() {
    return React.createElement(React.Fragment, {}, React.Children.map(this.props.children, child => {
      return React.cloneElement(child, {
        playbackReady: this.scriptingReady.bind(this),
        playbackDone: this.scriptingDone.bind(this),
        finishPlayback: this.finishScripting.bind(this),
        startPlayback: this.startScripting.bind(this),
        restartPlayback: this.restartScripting.bind(this),
        pausePlayback: this.pauseScripting.bind(this),
        editScripting: this.editScripting.bind(this),
        isPlaybackPaused: this.isPaused.bind(this),
        isPlaybackReset: this.isReset.bind(this),
        isEditing: this.isEditing.bind(this),
        isAutofilled: this.isAutofilled.bind(this),
        isScriptCompleted: this.isScriptCompleted.bind(this),
        resumePlayback: this.resumeScripting.bind(this),
        invalidatePlaybackPause: this.invalidatePauseState.bind(this),
        resetPlayback: this.resetScripting.bind(this),
        currentPause: this.currentPause.bind(this),
        TIMEOUT_BUFFER_RATIO,
      })
    }))
  }
}

const mapStateToProps = (state, props) => {
  const storeEditor = state.sandboxEditors.editorMap[props.editorId] || {}
  return {
    completionsMap: state.sandboxCompletions.completionsMap,
    cursorPos: storeEditor.cursorPos,
    cursorCaption: storeEditor.cursorCaption,
    cachedCompletions: state.cachedCompletions.cachedCompletions[props.activeScriptName]
  }
}

const mapDispatchToProps = dispatch => ({
  typeKey: (key, editorId, filename) => dispatch(typeKey(key, editorId, filename)),
  setCompletionIdx: (index, editorId) => dispatch(setCompletionIdx(index, editorId)),
  incrementCompletionIdx: (editorId) => dispatch(incrementCompletionIdx(editorId)),
  selectCompletion: (editorId) => dispatch(selectCompletion(editorId)),
  resetCompletionUI: (editorId) => dispatch(resetCompletionUI(editorId)),
  setBuffer: (buf, editorId, filename) => dispatch(setBuffer(buf, editorId, filename)),
  setCursor: (pos, editorId) => dispatch(setCursor(pos, editorId)),
  setCursorBlink: (blink, editorId) => dispatch(setCursorBlink(blink, editorId)),
  concatToBuffer: (str, editorId, filename) => dispatch(concatToBuffer(str, editorId, filename)),
  showCaption: (editorId, caption, completionCaptionPadding) => dispatch(showCaption(editorId, caption, completionCaptionPadding)),
  hideCaption: (editorId) => dispatch(hideCaption(editorId)),
  showOverlay: (editorId) => dispatch(showOverlay(editorId)),
  showCursorCaption: (editorId, caption, placement, marginTop, marginLeft, moves) => dispatch(showCursorCaption(editorId, caption, placement, marginTop, marginLeft, moves)),
  hideCursorCaption: (editorId) => dispatch(hideCursorCaption(editorId)),
  skipCompletions: (skip, editorId, filename) => dispatch(skipCompletions(skip, editorId, filename)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Scripter)