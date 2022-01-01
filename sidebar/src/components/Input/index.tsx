import React from 'react'

import styles from './index.module.css'

interface InputProps {
  className: string
  placeholder: string
  value: string
  status: InputStatus
  onSubmit: (e: React.FormEvent<HTMLFormElement>) => void
  onFocus: (e: React.FocusEvent<HTMLInputElement>) => void
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  onBlur: (e: React.FocusEvent<HTMLInputElement>) => void
  updateStatus: () => void
}

export enum InputStatus {
    None,
    Edit,
    Loading,
    Available,
    Unavailable,
}

export const Input: React.FunctionComponent<InputProps> = props => {
  const inputClasses = [styles.input]
  if (props.status === InputStatus.Unavailable) {
    inputClasses.push(styles.inputUnavailable)
  } else if (props.status === InputStatus.Edit) {
    inputClasses.push(styles.inputEdit)
  }
  let icon = ""
  const iconClasses = [styles.icon]
  if (props.status === InputStatus.Available) {
    icon = "✓"
  } else if (props.status === InputStatus.Unavailable) {
    icon = "✗"
  } else if (props.status === InputStatus.Edit) {
    icon = "➤"
    iconClasses.push(styles.iconClicky)
  } else if (props.status === InputStatus.Loading) {
    icon = "⋯"
    iconClasses.push(styles.iconWait)
  }
  return (
    <div>
      <div className={props.className}>
        <form className={inputClasses.join(' ')} onSubmit={props.onSubmit}>
          <input
            type="text"
            onFocus={props.onFocus}
            onChange={props.onChange}
            onBlur={props.onBlur}
            placeholder={props.placeholder}
            value={props.value}
          />
          <span className={iconClasses.join(' ')}>
            {icon}
          </span>
        </form>
      </div>
    </div>
  )
}
