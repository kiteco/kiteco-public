import * as React from "react"
import { connect } from "react-redux"
import { ThunkDispatch } from "redux-thunk"
import { AnyAction } from "redux"
import { CSSTransition } from "react-transition-group"
import * as hljs from "highlight.js"

import { openFile, State } from "../../../../store/related-code/related-code"
import { Block, Keyword, RelatedFile } from "../../../../store/related-code/api-types"
import { getIconForEditor, getReadableNameForEditor } from "../../../../utils/editorInfo"

import styles from './related-code-result.module.css'
import iconFolder from '../../assets/icon-folder.svg'

import '../../assets/hljs-light.css'
import '../../assets/hljs-dracula.css'
import '../../assets/hljs-high-contrast.css'

interface RelatedCodeResultProps {
  file_rank: number,
  openFile: (
    state: State,
    appPath: string,
    filename: string,
    line: number,
    file_rank: number,
    block_rank?: number,
  ) => Promise<void>,
  related_code: State,
  result: RelatedFile,
}

interface RelatedCodeResultState {
  showBlocks: boolean,
  showOpenHint: boolean,
  // What line to show the open hint on.
  // If 0, the open action was performed with no line.
  openHintLine: number,
  openHintMessage: string,
  dismissOpenHintTimeout?: NodeJS.Timeout,
}

/*
	RelatedCodeResult renders a single related file in a Related Code search
 */
class RelatedCodeResult extends React.Component<RelatedCodeResultProps, RelatedCodeResultState> {
  containerRef: React.RefObject<HTMLDivElement>
  constructor(props: any) {
    super(props)
    this.state = {
      showBlocks: false,
      showOpenHint: false,
      openHintLine: 0,
      openHintMessage: "",
    }
    this.containerRef = React.createRef<HTMLDivElement>()
  }

  componentDidUpdate() {
    this.highlightCodeblocks()
  }

  openFile = (line: number, file_rank: number, block_rank?: number,) => {
    const { related_code, result } = this.props

    let editorReadableName = getReadableNameForEditor(related_code.editor)
    if (!editorReadableName) {
      editorReadableName = "your editor"
    }
    let hintMessage = `Switch to ${editorReadableName} to view this ${line === 0 ? "file" : "code block"}`

    this.props.openFile(related_code, related_code.editor_install_path, result.file.absolute_path, line, file_rank, block_rank)
      .catch(r => {
        console.error(r)
        navigator.clipboard.writeText(result.file.absolute_path)
        hintMessage = `Filepath copied to clipboard → Paste into ${editorReadableName}`
      })
      .then(r => {
        this.setState(
          { showOpenHint: true, openHintLine: line, openHintMessage: hintMessage },
          () => {
            if (this.state.dismissOpenHintTimeout) {
              // if there's an old timeout, replace it with a fresh one
              clearTimeout(this.state.dismissOpenHintTimeout)
            }
            const timeout = setTimeout(() => {
              this.setState({ showOpenHint: false })
            }, 4000)
            this.setState({ dismissOpenHintTimeout: timeout })
          }
        )
      })
  }

  toggleView = () => {
    this.setState({ showBlocks: !this.state.showBlocks })
  }

  highlightCodeblocks = () => {
    if (this.containerRef && this.containerRef.current) {
      this.containerRef.current.querySelectorAll(`.${styles.codeblock_source} pre`)
        .forEach((node: any) => {
          hljs.highlightBlock(node)
        })
    }
  }

  render() {
    const { file_rank, result, related_code: { editor }} = this.props
    if (!result.file || !result.file.keywords) {
      console.error("null result or result keywords")
      return null
    }

    // insert zero-width spaces to break long paths after slashes
    // note: NodeJS doesn't support replaceAll, so we need to use a global search regex to replace all slashes
    const breakableFilepath =
      (result.relative_path + result.filename)
        .replace(/\//g, "/"+String.fromCharCode(0x200b))
        .replace(/\\/g, "\\"+String.fromCharCode(0x200b))

    let editorIcon = getIconForEditor(editor)
    if (!editorIcon) {
      editorIcon = iconFolder
    }

    return (
      <div ref={this.containerRef} className={styles.container}>
        <div className={styles.header}>
          <div className={styles.expand_collapse} onClick={this.toggleView} >
            {this.state.showBlocks ? "▾" : "▸"}
          </div>
          <div className={styles.title} onClick={this.toggleView} >
            {breakableFilepath}
          </div>
          <div className={styles.open_button} onClick={() => this.openFile(0, file_rank)}>
            <div>Open</div>
            {editor && <img className={styles.icon_plugin} src={editorIcon}/>}
          </div>
        </div>
        {
          this.state.showBlocks
            ? <BlockView
              blocks={result.file.blocks}
              file_rank={file_rank}
              {...this.state}
              editorIcon={editorIcon}
              openFile={this.openFile}
            />
            : <KeywordView
              keywords={result.file.keywords}
              {...this.state}
              editorIcon={editorIcon}
            />
        }
      </div>
    )
  }
}

interface BlockProps extends RelatedCodeResultState {
  blocks: Block[],
  file_rank: number,
  openFile: any
  editorIcon: any,
}

const BlockView = (props: BlockProps) => {
  const blocks = props.blocks.map((block: any, index: number) => {
    const block_rank = index + 1
    const lines = block.content.split("\n")

    let lineNumbers = ""
    for (let i = 0; i < lines.length; i++) {
      lineNumbers += (i + block.firstline).toString() + "\n"
    }

    return (
      <div key={block.content} className={styles.codeblock} onClick={() => props.openFile(block.firstline, props.file_rank, block_rank)}>
        <CSSTransition
          in={props.showOpenHint && props.openHintLine === block.firstline}
          timeout={{
            appear: 0,
            enter: 200,
            exit: 500,
          }}
          mountOnEnter
          unmountOnExit
          classNames={{
            enter: styles.hint_enter,
            enterActive: styles.hint_enter_active,
            exit: styles.hint_exit,
            exitActive: styles.hint_exit_active,
          }}
        >
          <div className={styles.codeline_hint}>
            <img className={styles.hint_icon} src={props.editorIcon}/>
            <div className={styles.hint_text}>
              {props.openHintMessage}
            </div>
          </div>
        </CSSTransition>
        <div className={styles.codeblock_linenums}>
          <pre>{lineNumbers}</pre>
        </div>
        <div className={styles.codeblock_source}>
          <pre>{block.content}</pre>
        </div>
      </div>
    )
  })
  return (
    <div className={styles.codeblock_list}>
      {blocks}
    </div>
  )
}


interface KeywordProps extends RelatedCodeResultState {
  keywords: Keyword[],
  editorIcon: any,
}

const KeywordView = (props: KeywordProps) => {
  const keywords = props.keywords.map((keyword: any, index: number) =>
    <div key={index} className={styles.keyword}>
      <code>{keyword.keyword}</code>
      {index === props.keywords.length - 1 ? "" : ","}
    </div>
  )
  return (
    <div className={styles.keyword_container}>
      <div className={styles.keyword_table}>
        <div className={styles.keyword_label}>
          Keywords
        </div>
        <div className={styles.keyword_words}>
          {keywords}
        </div>
      </div>
      <div className={styles.keyword_hint_container}>
        <CSSTransition
          in={props.showOpenHint}
          timeout={{
            appear: 0,
            enter: 200,
            exit: 500,
          }}
          mountOnEnter
          unmountOnExit
          classNames={{
            enter: styles.hint_enter,
            enterActive: styles.hint_enter_active,
            exit: styles.hint_exit,
            exitActive: styles.hint_exit_active,
          }}
        >
          <div className={styles.keyword_hint}>
            <img className={styles.hint_icon} src={props.editorIcon}/>
            <div className={styles.hint_text}>
              {props.openHintMessage}
            </div>
          </div>
        </CSSTransition>
      </div>
    </div>
  )
}

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    plugins: state.plugins,
    related_code: state.related_code,
  }
}

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  openFile: (
    state: State,
    appPath: string,
    filename: string,
    line: number,
    file_rank: number,
    block_rank?: number,
  ) => dispatch(openFile(state, appPath, filename, line, file_rank, block_rank)),
})

export default connect(mapStateToProps, mapDispatchToProps)(RelatedCodeResult)
