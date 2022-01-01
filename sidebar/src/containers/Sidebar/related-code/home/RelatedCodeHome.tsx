import * as React from "react"

import styles from './related-code-home.module.css'

import { Domains } from "../../../../utils/domains"

const electron = window.require('electron')

/*
	RelatedCodeHome is the component that manages the main content in the Related Code Dashboard when there is no active search
 */
const RelatedCodeHome = () => (
  <div className={styles.container}>
    <div className={`${styles.hint} ${styles.hint_icon_graph} showup__animation`}>
      <div className={styles.hint_paragraph}>
        The Copilot can help you find code blocks related to the piece of code that you are working with.
      </div>
      <div className={styles.hint_paragraph}>
        Use this feature to quickly navigate your codebase to find groups of interconnected code.
      </div>
      <div className={styles.hint_link} onClick={() => electron.shell.openExternal(`https://${Domains.Help}/article/147-find-related-code-in-the-copilot`)}>
        Learn more about how to use this feature.
      </div>
    </div>
    <div className={`${styles.hint} ${styles.hint_icon_atom_sublime_vscode} showup__animation showup__animation--delay`}>
      <div className={styles.hint_paragraph}>
        In Atom. Sublime, and VSCode, run the command <span>Kite: Find Related Code From File</span> from the command palette.
      </div>
    </div>
    <div className={`${styles.hint} ${styles.hint_icon_jetbrains} showup__animation showup__animation--delay-2`}>
      <div className={styles.hint_paragraph}>
        In JetBrains editors, run the action <span>Kite: Find Related Code From File</span> from the action finder.
      </div>
    </div>
    <div className={`${styles.hint} ${styles.hint_icon_vim} showup__animation showup__animation--delay-3`}>
      <div className={styles.hint_paragraph}>
        In Vim, run the command <span>:KiteFindRelatedCodeFromFile</span> while in normal mode.
      </div>
    </div>
  </div>
)

export default RelatedCodeHome
