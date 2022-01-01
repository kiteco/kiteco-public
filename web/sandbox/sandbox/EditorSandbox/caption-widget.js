/**
 * completions widget is going to always show below the given
 * use the "above" property in the "options object to pass in"
 * widgetObj = doc.addLineWidget(line, node, options)
 * widgetObj.clear()
 */

export const createCaptionWidget = (caption, addCompletionPadding) => {
  const captionLines = caption.split('\n')
  const widget = document.createElement('div')
  widget.className = "CodeMirror-caption-widget"
  if(addCompletionPadding) {
    const paddingNode = document.createElement('div')
    paddingNode.className = "CodeMirror-caption-widget--completion-padding"
    widget.appendChild(paddingNode)
  }
  captionLines.forEach(line => {
    const lineNode = document.createElement('p')
    lineNode.className = "CodeMirror-caption-widget--line"
    const content = document.createTextNode(line)
    lineNode.appendChild(content)
    widget.appendChild(lineNode)
  })
  return widget
}

export const createCursorCaptionWidget = (caption, { marginTop="", marginLeft="", afterClass="" }) => {
  const captionLines = caption.split('\n')
  const widget = document.createElement('div')
  widget.className = `CodeMirror-cursor-caption-widget ${afterClass ? `${afterClass}` : ''}`
  captionLines.forEach(line => {
    const textNode = document.createElement('p')
    const content = document.createTextNode(line)
    textNode.appendChild(content)
    widget.appendChild(textNode)
  })
  if(marginLeft) {
    widget.style.marginLeft = marginLeft
  }
  if(marginTop) {
    widget.style.marginTop = marginTop
  }
  return widget
}