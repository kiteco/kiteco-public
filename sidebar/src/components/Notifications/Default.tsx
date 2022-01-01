import React from 'react'
import styles from './Default.module.css'

interface Props {
  title: string
  text: string
  noDismiss: boolean | undefined
  buttons: Button[]
  dismiss: () => void
}

interface Button {
  text: string
  noDismiss: boolean | undefined
  onClick: () => void
}

const Default = ({ noDismiss, dismiss, title, text, buttons }: Props) =>
  <section className={styles.notification}>
    <header>
      <h1>
        {title}
      </h1>
      {
        !noDismiss &&
        <button onClick={dismiss}>
          Hide
        </button>
      }
    </header>

    <div>
      <p>
        {text}
      </p>

      {buttons && buttons.map(btn =>
        <button onClick={() => {
          !btn.noDismiss && dismiss()
          btn.onClick()
        }}>
          {btn.text}
        </button>
      )}
    </div>
  </section>

export default Default
