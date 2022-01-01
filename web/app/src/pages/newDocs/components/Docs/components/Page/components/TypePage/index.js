import React from 'react'
import { connect } from 'react-redux'
import { Helmet } from 'react-helmet'

import Description from '../Description'
import Promo from '../Promo'
import HowToExamples from '../HowToExamples'
import Parameters from '../Parameters'
import Kwargs from '../Kwargs'
import ReturnValue from '../ReturnValue'
import HowOthers from '../HowOthers'

import { type as meta } from '../../../util/meta-description'

const TypePage = ({
  doc,
  name,
  type,
  loggedIn,
  style,
  language,
  location,
}) => {
  return <div className='items'>
    <Helmet>
      <title>{name} - {doc.ancestors.length > 0 ? `${doc.ancestors[0].name} - ` : ''}Python documentation - Kite</title>
      <meta
        name="description"
        content={meta({ doc, name })}
      />
    </Helmet>
    <div className='item'>
      <div className={`intro fixed-height-intro fixed-height-intro-spacing ${style.extraIntroClass}`}></div>
      <div className='docs'>
        { (doc.description_html || doc.documentation_str)
          && <Description
            language={language}
            description_html={doc.description_html}
            docString={doc.documentation_str}
            defaultExpanded={!(doc.exampleIds && doc.exampleIds.length)}
          />
        }
        <HowToExamples language={language} exampleIds={doc.exampleIds} answers_links={doc.answers_links} />
      </div>
      <div className='extra'>
        { !loggedIn &&
          <Promo/>
        }
        {doc.parameters && doc.parameters.length > 0 && <Parameters name={name} parameters={doc.parameters} />}
        {doc.kwargs && doc.kwargs.length > 0 && <Kwargs kwargs={doc.kwargs}/>}
        {doc.returnValues && doc.returnValues.length > 0 && <ReturnValue returnValues={doc.returnValues}/>}
        {doc.patterns && doc.patterns.length > 0 && <HowOthers patterns={doc.patterns} />}
      </div>
    </div>
  </div>
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  style: state.style,
  location: state.router.location,
})

export default connect(mapStateToProps)(TypePage)
