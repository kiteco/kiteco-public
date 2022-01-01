import React from 'react'
import { connect } from 'react-redux'
import { Helmet } from 'react-helmet'

import Description from '../Description'
import HowToExamples from '../HowToExamples'
import Promo from '../Promo'

const ModulePage = ({ doc, name, type, loggedIn, style, language, location }) => {
  return (
    <div className='items'>
      <Helmet>
        <title>{name} - {doc.ancestors.length > 0 ? `${doc.ancestors[0].name} - ` : ''}Python documentation - Kite</title>
    {/*
        <meta
          name="description"
          content={meta({ doc, name })}
        />
    */}
      </Helmet>
      <div className='item'>
        <div className={`intro fixed-height-intro fixed-height-intro-spacing ${style.extraIntroClass}`}></div>
        <div className='docs'>
          { (doc.description_html || doc.documentation_str) &&
            <Description
              language={language}
              description_html={doc.description_html}
              docString={doc.documentation_str}
              defaultExpanded={true}
            />
          }
        </div>
        <div className='extra'>
          { !loggedIn &&
            <Promo/>
          }
          <HowToExamples language={language} exampleIds={doc.exampleIds} answers_links={doc.answers_links} />
        </div>
      </div>
    </div>
  )
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  style: state.style,
  location: state.router.location,
})

const mapDispatchToProps = dispatch => ({

})

export default connect(mapStateToProps, mapDispatchToProps)(ModulePage)
