import React from 'react'

import DownloadButton from '../../../components/DownloadButton'
import SignInForm from '../../SignInForm'

import '../assets/promotion.css'

const NewUserPromo = ({ os }) => {
  return <div className="docs__new-user">
    <div className="docs__new-user__text">
      { os === "linux"
        ? "Sign up for Kite to enjoy uninterrupted access"
        : "Get Python completions, documentation, code usages, examples and more all within your editor"
      }
    </div>
      { os === "linux"
        ? <SignInForm source="new"/>
        : <DownloadButton
            os={os}
            className="docs__download-button"
          />
      }
  </div>
}

export default NewUserPromo
