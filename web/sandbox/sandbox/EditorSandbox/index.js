import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import OnVisible from 'react-on-visible'
import uuid from 'uuid/v4'

import SandboxOverlay from '../SandboxOverlay'
//import RestartButton from '../Restart'
//import SandboxSocialSharing from '../SandboxSocialSharing'

import { fetchCompletions, LOAD_COMPLETIONS_FAILED, COMPLETIONS_FILE_TOO_LARGE } from '../../actions/sandbox-completions'
import {
  setBuffer,
  completionSelected,
  completionMoved,
  editorFocused,
  registerEditor,
  deleteEditor,
  cursorSet,
  completionsClosed,
  setCursorPos,
  hideHoverOverlay,
  showHoverOverlay,
  hideOverlay,
  setCompletionsBoxDimensions,
} from '../../actions/sandbox-editors'

import { createCaptionWidget, createCursorCaptionWidget } from './caption-widget'
import { track } from '../../utils/analytics'
import { onRemove } from '../../utils/dom'


import {Controlled as CodeMirror} from 'react-codemirror2'
import 'codemirror/lib/codemirror.css';
import './themes/kite-dark.css';
import './themes/kite-light.css';
import 'codemirror/addon/hint/show-hint.css';
import 'codemirror/addon/selection/active-line.js';
import './assets/editor-sandbox.css'
import { RIGHT_CURSOR_CAPTION_PLACEMENT, BOTTOM_CURSOR_CAPTION_PLACEMENT } from '../scripts';

const CM = require('codemirror')

require('../../utils/sandbox-completions')(CM)
require('codemirror/addon/edit/closebrackets')
require('codemirror/addon/edit/matchbrackets')
require('codemirror/mode/python/python')

const HOVER_THRESHOLD = 100 //ms



class EditorSandbox extends React.Component {

  constructor(props) {
    super(props)
    this.editor = null
    this.editorId = uuid()
    this.captionWidget = null
    this.userChangeCount = 0
    this.userCompletionsCount = 0
    this.wrapperRef = React.createRef()

    this.state = {
      lastCompletion: "",
      activelyEditing: false,
      showSocialSharing: false,
      editingInitialized: false,
    }
  }

  componentWillUnmount() {
    //send change count analytics up
    track({
      event: "web-sandbox: number of user changes made",
      props: {
        changes_made: this.userChangeCount,
        page: window.location.pathname,
        source: "wordpress"
      }
    })
    track({
      event: "web-sandbox: number of completions selected",
      props: {
        num_completions_used: this.userCompletionsCount,
        page: window.location.pathname,
        source: "wordpress"
      }
    })
  }

  // WordPress will detach the DOM node inserted by the HTMLSection in the ACF interface after some amount of time
  // The detachment is somewhat insidious, as a copy of the structure replaces it
  // The root reason why this occurs is opaque; however, it is what happens
  // Thus, we need to listen for the detachment, and reattach when detachment occurs
  componentDidMount() {
    this.domNode = ReactDOM.findDOMNode(this)
    const detachmentHandler = () => {
      const wpWrapperEl = document.getElementById('kite-web-sandbox')
      const hangingCMEl = document.getElementById('CodeMirror-wrapper')
      wpWrapperEl.replaceChild(this.domNode, hangingCMEl)
      // make sure to re-setup detachment observer 
      onRemove(this.domNode, detachmentHandler)
    }
    onRemove(this.domNode, detachmentHandler)
  }

  onChange(editor, data, value) {
    //don't show after making a completion
    if(data.origin !== 'complete') {
      editor.showHint()
    }
    //data.origin is undefined in non-completion Scripter generated changes
    if(data.origin && !(this.props.isEditing && !this.props.isEditing())) {
      if(!this.state.activelyEditing) {
        this.setState({ activelyEditing: true })
      }
      this.userChangeCount++
    }
    if(!data.origin && this.props.isEditing && !this.props.isEditing()) {
      if(this.state.activelyEditing) {
        this.setState({ activelyEditing: false })
      }
    }
  }

  componentDidUpdate(prevProps) {
    if(this.editor) {
      //shouldselect case
      if(!prevProps.shouldSelectCompletion && this.props.shouldSelectCompletion) {
        this.editor.selectActive()
        this.props.completionSelected(this.editorId)
      }
      //shouldclose case
      if(!prevProps.shouldCloseCompletions && this.props.shouldCloseCompletions) {
        this.editor.closeHint()
        this.props.completionsClosed(this.editorId)
      }
      //should move case
      if(this.props.completionIndex >= 0 && this.props.shouldMoveCompletion){
        this.editor.moveActive(1)
        this.props.completionMoved(this.editorId)
      }
      //should focus editor case
      if(!prevProps.shouldFocus && this.props.shouldFocus) {
        this.editor.focus()
        this.props.editorFocused(this.editorId)
      }
      //should blink cursor case
      if(prevProps.cursorBlink !== this.props.cursorBlink) {
        this.editor.scriptCursorBlinking(this.props.cursorBlink)
      }
      //should show caption case
      if(!prevProps.showCaption && this.props.showCaption) {
        this.addCaptionWidget()
        // will want to recalculate hint position here
        this.editor.showHint()
      }
      //should show cursorCaption case
      if(!prevProps.showCursorCaption && this.props.showCursorCaption) {
        this.addCursorCaptionWidget()
      }
      //tab switch case
      if(prevProps.activeFile !== this.props.activeFile) {
        this.editor.showHint()
      }
      //should hide caption case
      if(prevProps.showCaption && !this.props.showCaption) {
        // will want to recalculate hint pos here
        // hide hint; animate out; show hint
        this.captionWidget && this.captionWidget.clear()
        this.editor.showHint()
      }
      //should hide cursor caption case
      if(prevProps.showCursorCaption && !this.props.showCursorCaption) {
        this.removeCursorCaptionWidget()
        //do I want to recalclulate the hint pos here?
      }
      //should set cursor
      if(!prevProps.cursor && this.props.cursor) {
        this.editor.getDoc().setCursor(this.props.cursor)
        this.props.cursorSet()
      }
      /* //should showOverlay (could definitely clean up a little bit)
      if(!prevProps.shouldShowOverlay && this.props.shouldShowOverlay) {
        this.setState({ showOverlay: true })
      }
      if(prevProps.shouldShowOverlay && !this.props.shouldShowOverlay) {
        this.setState({ showOverlay: false })
      } */
      //should showHoverOverlay (could also stand to clean up a bit)
      //may not actually need store usage here... could be purely state
      /* if(!prevProps.shouldShowHoverOverlay && this.props.shouldShowHoverOverlay) {
        this.setState({ showHoverOverlay: true })
      }
      if(prevProps.shouldShowHoverOverlay && !this.props.shouldShowHoverOverlay) {
        this.setState({ showHoverOverlay: false })
      } */
    }
  }

  calculateCursorCaptionPos() {
    //if no completionsBoxDimensions, then use cursorCoords(false)
    if(this.props.completionsBoxDimensions && this.props.completionsBoxDimensions.hasOwnProperty('top') && this.props.completionsBoxDimensions.hasOwnProperty('left')) {
      const { cursorCaptionPlacement, completionsBoxHighlightLevel } = this.props
      let { left, top, width, height } = this.props.completionsBoxDimensions
      switch(cursorCaptionPlacement) {
        case RIGHT_CURSOR_CAPTION_PLACEMENT:
          const highlightHeight = (completionsBoxHighlightLevel - 1) * 24 //font size in pixels
          return { left, top, width, height: highlightHeight }
        case BOTTOM_CURSOR_CAPTION_PLACEMENT:
          return { left, top, height }
        default:
          break
      }
      return {
        left,
        top,
        width,
        height,
      }
    } else {
      const { left, top } = this.editor.cursorCoords(false)
      return { left, top }
    }
  }

  addCursorCaptionWidget() {
    const { cursorCaption, cursorCaptionMarginLeft, cursorCaptionMarginTop, cursorCaptionAfterClass } = this.props
    if(cursorCaption) {
      const { left, top, width=0, height=0 } = this.calculateCursorCaptionPos()
      this.cursorCaptionWidget = createCursorCaptionWidget(cursorCaption, {
        marginLeft: `${width + 10 + cursorCaptionMarginLeft}px`, //10px gives a nice buffer
        marginTop: `${height + cursorCaptionMarginTop}px`,
        afterClass: cursorCaptionAfterClass,
      })
      const pos = this.editor.coordsChar({ left, top }, "local");
      //add widget
      this.editor.addWidget(pos, this.cursorCaptionWidget, true)
    }
  }

  removeCursorCaptionWidget() {
    // find ref on parent, remove
    this.cursorCaptionWidget &&
     this.cursorCaptionWidget.parentNode &&
      this.cursorCaptionWidget.parentNode.removeChild(this.cursorCaptionWidget)
  }

  addCaptionWidget() {
    if(this.props.caption) {
      //add extra caption padding processing here
      const doc = this.editor.getDoc()
      const line = doc.lastLine()
      const widget = createCaptionWidget(this.props.caption, this.props.completionCaptionPadding)
      this.captionWidget = doc.addLineWidget(line, widget)
    }
  }

  shouldUseCachedCompletions(cursorPos) {
    return this.props.isEditing && !this.props.isEditing()
      && this.props.cachedCompletions && this.props.cachedCompletions[cursorPos]
  }

  shouldUseRemoteCompletions() {
    if(this.props.isEditing) {
      return this.props.isEditing()
    }
    return true
  }

classFromKind(kind) {
    switch(kind) {
      case '':
        return 'kind__unknown'
      case 'module':
      case 'type':
      case 'function':
      case 'unknown':
      case 'instance':
      case 'keyword':
        return `kind__${kind}`
      default:
        return 'kind__instance'
    }
  }

  editorWillUnmount(editor) {
    this.editor && this.editor.closeHint()
    this.props.finishPlayback && this.props.finishPlayback()
    this.props.deleteEditor(this.editorId)
  }

  editorDidMount(editor) {
    this.editor = editor
    this.props.registerEditor(this.editorId)
    this.props.setIdCb && this.props.setIdCb(this.editorId)

    const renderFn = (el, self, data) => {
      const displayText = document.createTextNode(data.displayText)
      const displayNode = document.createElement('span')
      displayNode.className = 'completion-display'
      displayNode.appendChild(displayText)
      //const hintNode = document.createElement('span')
      //hintNode.className = 'completion-hint'
      //const hintText = document.createTextNode(data.hintText)
      //hintNode.appendChild(hintText)
      el.appendChild(displayNode)
      //ADD class name based on hintText
      el.classList.add(this.classFromKind(data.hintText))
      //el.appendChild(hintNode)
    }

    const shownCompletionsHandler = (widget) => {
      const { top, left, width, height } = widget.hintsWrapper.getBoundingClientRect()
      const { top: curTop,
              left: curLeft,
              width: curWidth,
              height: curHeight } = this.props.completionsBoxDimensions || {}
      if(top !== curTop || left !== curLeft || width !== curWidth || height !== curHeight) {
        this.props.setCompletionsBoxDimensions(this.editorId, { top, left, width, height })
      }
    }

    const isProbablyMultiToken = completion => {
      // some hacky approximation
      return completion.includes('.') || completion.includes('(')
    }

    const closeCompletionsHandler = () => {
      this.props.setCompletionsBoxDimensions(this.editorId, {})
    }

    const pickCompletionsHandler = (completionObj) => {
      const completion = completionObj
        ? completionObj.displayText
        : ""
      if((completion && isProbablyMultiToken(completion)) || completion === "") {
        this.setState({ lastCompletion: completion })
      }
      this.setState({ userCompletionsCount: this.state.userCompletionsCount + 1 })
      track({
        event: "web-sandbox: completion selected",
        props: {
          completion_used: completionsObj.symbol ? completionObj.symbol.id : completionObj.text,
          page: window.location.pathname,
          source: "wordpress"
        }
      })
    }

    const hint = (cm, cb, opts) => {
      const cursor = cm.doc.getCursor()
      const cursorPos = cm.doc.indexFromPos(cursor)
      const cursorToken = cm.getTokenAt(cursor)
      this.props.setCursorPos(this.editorId, cursorPos)

      let from = {
        line: cursor.line,
        ch: cursorToken.start
      }
      let to = {
        line: cursor.line,
        ch: cursorToken.end
      }
      //e.g. a '.' or a ' '
      if(!cursorToken.type) {
        from.ch = to.ch
      }
      let timeout
      if(this.props.isEditing && !this.props.isEditing()) {
        if(this.props.currentPause) {
          timeout = this.props.currentPause()
          if(timeout && this.props.TIMEOUT_BUFFER_RATIO) {
            timeout -= Math.floor(timeout * this.props.TIMEOUT_BUFFER_RATIO)
          }
        }
      }
      // test for editMode here
      if(this.props.skipCompletions) {
        cb()
      } else if(this.shouldUseCachedCompletions(cursorPos)) {
        const list = this.props.cachedCompletions[cursorPos].map(completion => ({
          text: completion.insert,
          displayText: completion.display,
          hintText: completion.hint,
          render: renderFn,
        }))
        const completionsObj = {
          list,
          from,
          to,
        }
        CM.on(completionsObj, "close", closeCompletionsHandler)
        CM.on(completionsObj, "shown", shownCompletionsHandler)
        cb(completionsObj)
      } else if(this.shouldUseRemoteCompletions()) {
        this.props.fetchCompletions(cm.doc.getValue(), cursorPos, this.editorId, this.props.activeFile, timeout)
        .then(completions => {
          if(completions === LOAD_COMPLETIONS_FAILED || completions === COMPLETIONS_FILE_TOO_LARGE) {
            cb()
          } else if (completions) {
            const list = completions.map(completion => {
              return {
                text: completion.insert,
                displayText: completion.display,
                hintText: completion.hint,
                symbol: completion.symbol,
                render: renderFn,
              }
            })
            const completionsObj = {
              list,
              from,
              to,
            }
            CM.on(completionsObj, "pick", pickCompletionsHandler)
            CM.on(completionsObj, "close", closeCompletionsHandler)
            CM.on(completionsObj, "shown", shownCompletionsHandler)
            cb(completionsObj)
          } else {
            cb()
          }
        })
      } else {
        cb()
      }
    }

    hint.async = true
    editor.options.hintOptions = {
      hint,
      completeSingle: false,
      closeOnUnfocus: false,
    }
    this.props.playbackReady && this.props.playbackReady()
    this.onVisiblity()
  }

  onBeforeChange(editor, data, value) {
    this.props.setBuffer(value, this.editorId, this.props.activeFile)
  }

  onFocus() {
    if(this.editor) {
      this.editor.scriptCursorBlinking(false)
    }
    this.captionWidget && this.captionWidget.clear()
    if(this.props.isEditing && !this.props.isEditing()) {
      // then autofill
      this.props.finishPlayback && this.props.finishPlayback()
    }
  }

  onVisiblity(visible) {
    if(this.editor) {
      this.editor.scriptCursorBlinking(true)
    }
    if(this.props.startPlayback) {
      setTimeout(() => {
        this.props.startPlayback()
      }, 500)
    }
  }

  isInViewPort(offset = 0) {
    if(!this.editorEl) {
      return false
    }
    const top = this.editorEl.getBoundingClientRect().top
    return (top + offset) >= 0 && (top - offset) <= window.innerHeight
  }

  cursorClass = () => {
    return this.props.isEditing && this.props.isEditing()
      ? ' cursor-text'
      : ' cursor-default'
  }

  overlayClickCb = () => {
    track({
      event: "web-sandbox: edit mode entered",
      props: {
        page: window.location.pathname,
        source: "wordpress"
      }
    })
    this.setState({ showSocialSharing: true })
    setImmediate(() => {
      this.setState({ editingInitialized: true })
    })
  }

  restartCb = () => {
    this.setState({ showSocialSharing: false })
    this.props.restartPlayback()
  }

  shouldShowSocialSharingBtns = () => {
    return this.props.isEditing
     && this.props.isEditing()
     && this.state.showSocialSharing
  }

  getLineCount = () => {
    return this.editor && this.editor.doc
      ? this.editor.doc.lineCount()
      : 0
  }

  getLineHeight = () => {
    return this.editor && this.editor.defaultTextHeight()
  }

  render() {
    const className = this.props.className
      ? this.props.className
      : `CodeMirror-wrapper--default ${this.props.theme ? `${this.props.theme}` : ''}`

    return (
      /* restore onVisibility */
        <div
          ref={this.wrapperRef}
          className={`${className}${this.cursorClass()}`}
          id="CodeMirror-wrapper"
        >
          {this.props.isEditing && !this.props.isEditing() && <SandboxOverlay
            editorId={this.editorId}
            editScripting={this.props.editScripting}
            isEditing={this.props.isEditing}
            cursorClass={'cursor-default'}
            clickCb={this.overlayClickCb}
          />}
          {/* !this.state.showOverlay && this.state.showHoverOverlay
            && <SandboxHoverOverlay
              editorId={this.editorId}
              invalidatePlaybackPause={this.props.invalidatePlaybackPause}
              /> */
          }
          <CodeMirror
            options={{
              mode: 'python',
              theme: this.props.theme || 'kite-dark',
              lineNumbers: true,
              readOnly: false,
              styleActiveLine: true,
              scrollbarStyle: null,
            }}
            value={this.props.buffer}
            editorDidMount={this.editorDidMount.bind(this)}
            editorWillUnmount={this.editorWillUnmount.bind(this)}
            onBeforeChange={this.onBeforeChange.bind(this)}
            onChange={this.onChange.bind(this)}
            onFocus={this.onFocus.bind(this)}
          />
          {/* this.shouldShowSocialSharingBtns() && <SandboxSocialSharing
            lastCompletion={this.state.lastCompletion}
            theme={this.props.theme}
            shouldCollapse={this.state.activelyEditing}
            lineCount={this.getLineCount()}
            editingInitialized={this.state.editingInitialized}
            codeLineHeight={this.getLineHeight()}
          /> */}
          {/* <RestartButton
            restartPlayback={this.restartCb}
          /> */}
        </div>
      
    )
  }
}

const mapStateToProps = (state, props) => {
  const storeEditor = state.sandboxEditors.editorMap[props.editorId] || { bufferMap: {}, completionsBoxDimensions: {} }
  return {
    cachedCompletions: state.cachedCompletions.cachedCompletions[props.activeFile],
    completions: state.sandboxCompletions.completionsMap[props.editorId+props.activeFile],
    skipCompletions: state.sandboxCompletions.skipCompletionsMap[props.editorId+props.activeFile],
    buffer: storeEditor.bufferMap[props.activeFile],
    shouldSelectCompletion: storeEditor.shouldSelectCompletion,
    shouldMoveCompletion: storeEditor.shouldMoveCompletion,
    shouldFocus: storeEditor.shouldFocusEditor,
    completionIndex: storeEditor.completionIndex,
    showCaption: storeEditor.showCaption,
    showCursorCaption: storeEditor.showCursorCaption,
    cursorCaption: storeEditor.cursorCaption,
    cursorCaptionPlacement: storeEditor.cursorCaptionPlacement,
    cursorCaptionMarginTop: storeEditor.cursorCaptionMarginTop,
    cursorCaptionMarginLeft: storeEditor.cursorCaptionMarginLeft,
    cursorCaptionAfterClass: storeEditor.cursorCaptionAfterClass,
    caption: storeEditor.caption,
    cursor: storeEditor.cursor,
    cursorBlink: storeEditor.cusorBlink,
    completionCaptionPadding: storeEditor.completionCaptionPadding,
    shouldCloseCompletions: storeEditor.shouldCloseCompletions,
    shouldShowOverlay: storeEditor.shouldShowOverlay,
    shouldShowHoverOverlay: storeEditor.shouldShowHoverOverlay,
    completionsBoxDimensions: storeEditor.completionsBoxDimensions,
    completionsBoxHighlightLevel: storeEditor.completionsBoxHighlightLevel,
  }
}

const mapDispatchToProps = dispatch => ({
  fetchCompletions: (text, cursorBytes, id, filename, timeout) => dispatch(fetchCompletions(text, cursorBytes, id, filename, timeout)),
  setBuffer: (buffer, id, filename) => dispatch(setBuffer(buffer, id, filename)),
  completionSelected: (id) => dispatch(completionSelected(id)),
  completionMoved: (id) => dispatch(completionMoved(id)),
  completionsClosed: (id) => dispatch(completionsClosed(id)),
  editorFocused: (id) => dispatch(editorFocused(id)),
  registerEditor: (id) => dispatch(registerEditor(id)),
  deleteEditor: (id) => dispatch(deleteEditor(id)),
  cursorSet: (id) => dispatch(cursorSet(id)),
  setCursorPos: (id, pos) => dispatch(setCursorPos(id, pos)),
  hideHoverOverlay: (id) => dispatch(hideHoverOverlay(id)),
  showHoverOverlay: (id) => dispatch(showHoverOverlay(id)),
  hideOverlay: (id) => dispatch(hideOverlay(id)),
  setCompletionsBoxDimensions: (id, dimensions) => dispatch(setCompletionsBoxDimensions(id, dimensions)),
})

export default connect(mapStateToProps, mapDispatchToProps)(EditorSandbox)
