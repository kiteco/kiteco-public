import React, { useState } from 'react'
import { Redirect } from "react-router"

import JSLogo from '../assets/icon-javascript.svg'
import HTMLLogo from '../assets/icon-html5.svg'
import JavaLogo from '../assets/icon-java.svg'
import CLogo from '../assets/icon-c.png'
import GoLogo from '../assets/icon-go-gopher.png'
import RCDocsDashboardParagraph from "../../RemoteContent/RCDocsDashboardParagraph";

enum Langs {
  Javascript = 'javascript',
  HTML = 'html',
  Java = 'java',
  C = 'c',
  Go = 'go',
}

interface DocsHomeProps {
  identifier?: string
}

const DocsHome = (props: DocsHomeProps) => {
  const [hoveredLang, setHoveredLang] = useState<Langs | null>(null)
  const clearHoveredLang = () => setHoveredLang(null)

  if (props.identifier) {
    return (
      <Redirect to={`/docs/${props.identifier}`} />
    )
  }

  return (
    <div className="docs-page__root">
      <h4 className="showup__animation">
        Welcome to the Kite Copilot! <div className="docs-page__root__welcome-graphic"/>
      </h4>
      <RCDocsDashboardParagraph/>
      <div className="docs-page__root__hint docs-page__root__hint--python-logo showup__animation showup__animation--delay">
        <div className="docs-page__root__hint-paragraph">
          The Copilot automatically shows documentation as you type in your editor when the above "following cursor" button is enabled. You can also search for docs in the search bar.
        </div>
        <div>
          Kite also suggests Python completions as you code in your editor.
        </div>
      </div>

      <div className="docs-page__hint-container showup__animation showup__animation--delay-2">
        <div className="docs-page__root__hint docs-page__root__hint--javascript-logo">
          Kite can suggest completions inside your editor for many other languages. Hover your mouse over the language icons to see what file types Kite supports.
        </div>
        <div className="docs-page__hint__footer">
          <LangIcon lang={Langs.Javascript} onMouseEnter={setHoveredLang} onMouseLeave={clearHoveredLang} />
          <LangIcon lang={Langs.HTML} onMouseEnter={setHoveredLang} onMouseLeave={clearHoveredLang} />
          <LangIcon lang={Langs.Java} onMouseEnter={setHoveredLang} onMouseLeave={clearHoveredLang} />
          <LangIcon lang={Langs.C} onMouseEnter={setHoveredLang} onMouseLeave={clearHoveredLang} />
          <LangIcon lang={Langs.Go} onMouseEnter={setHoveredLang} onMouseLeave={clearHoveredLang} />
        </div>
        <HoveredLang lang={hoveredLang}/>
      </div>
    </div>
  )
}

const contentFromLang = {
  [Langs.Javascript]: {
    icon: {
      src: JSLogo,
      alt: "JavaScript logo",
    },
    hover: {
      title: 'Javascript',
      msg: 'Kite provides completions in .js, .jsx, .ts, .tsx, and .vue files. Kite does not have documentation for these files.',
    },
  },
  [Langs.HTML]: {
    icon: {
      src: HTMLLogo,
      alt: "JavaScript logo",
    },
    hover: {
      title: 'HTML & CSS',
      msg: 'Kite provides completions in .html, .css, and .less files. Kite does not have documentation for these files.',
    },
  },
  [Langs.Java]: {
    icon: {
      src: JavaLogo,
      alt: "Java logo",
    },
    hover: {
      title: 'Java',
      msg: 'Kite provides completions in .java, .kt, and .scala files. Kite does not have documentation for these files.',
    },
  },
  [Langs.C]: {
    icon: {
      src: CLogo,
      alt: "C logo",
    },
    hover: {
      title: 'C-Style Languages',
      msg: 'Kite provides completions in .c, .cc, .cpp, .cs, .h, .hpp, and .m files. Kite does not have documentation for these files.',
    },
  },
  [Langs.Go]: {
    icon: {
      src: GoLogo,
      alt: "Go logo",
    },
    hover: {
      title: 'Go',
      msg: 'Kite provides completions in .go files. Kite does not have documentation for these files.',
    },
  },
}

interface LangIconProps {
  lang: Langs,
  onMouseEnter: (l: Langs) => void,
  onMouseLeave: () => void,
}

const LangIcon = (props: LangIconProps) => {
  const { icon: { src, alt }} = contentFromLang[props.lang]
  return (
    <img
      className="docs-page__lang-icon"
      src={src}
      alt={alt}
      onMouseEnter={() => props.onMouseEnter(props.lang)}
      onMouseLeave={props.onMouseLeave}
    />
  )
}

const HoveredLang = (props: { lang: Langs | null }) => {
  if (props.lang === null) {
    return null
  }

  const { hover: { title, msg }} = contentFromLang[props.lang]
  return (
    <div className="sidebar__tooltip">
      <div className="sidebar__tooltip__title">
        {title}
      </div>
      <div className="sidebar__tooltip__paragraph">
        {msg}
      </div>
    </div>
  )
}

export default DocsHome
