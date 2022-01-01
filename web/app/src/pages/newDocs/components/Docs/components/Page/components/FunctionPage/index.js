import React from 'react'
import { connect } from 'react-redux'
import { Helmet } from 'react-helmet'

import { BreadcrumbWrapper, Breadcrumbs, StarHeader, TypeIndicator } from '../../../util/Titles'
import { func as meta } from '../../../util/meta-description'
import Description from '../Description'
import HowToExamples from '../HowToExamples'
import Parameters from '../Parameters'
import Kwargs from '../Kwargs'
import ReturnValue from '../ReturnValue'
import HowOthers from '../HowOthers'
import Promo from '../Promo'

import { addStarred, removeStarred } from '../../../../../../../../redux/actions/starred'
import DocsSpinner from '../../../util/DocsSpinner'

const FunctionPage = ({
  doc,
  name,
  type,
  language,
  loggedIn,
  addStarred,
  removeStarred,
  starred,
  location,
  loading }) => {
  const isStarred = starred.starredMap[location.pathname] ? true : false
  const clickHandler = isStarred ? removeStarred(location.pathname) : addStarred(name, location.pathname, 'identifier')
  return (
    <div className='items' data-mock-page-type='FUNCTION'>
      <Helmet>
        <title>{name} - {doc.ancestors && doc.ancestors.length > 0 ? `${doc.ancestors[0].name} - ` : ''}Python documentation - Kite</title>
        <meta name="description" content={meta({ doc, name })}/>
      </Helmet>
      <div className='item'>
        { loading.isDocsLoading && <div className='intro'>
          <DocsSpinner />
        </div> }
        { !loading.isDocsLoading && <div className='intro'>
          <BreadcrumbWrapper>
            <Breadcrumbs items={doc.ancestors} />
            <StarHeader
              moduleName={name}
              type='function'
              isStarred={isStarred}
              clickHandler={clickHandler}
              path={location.pathname}
              position={doc.ancestors.length + 1}
            />
          </BreadcrumbWrapper>
          <TypeIndicator type={type}/>
        </div> }

        <div className='docs'>
          { (doc.description_html || doc.documentation_str)
            && <Description
              language={language}
              description_html={doc.description_html}
              docString={doc.documentation_str}
              defaultExpanded={!(doc.exampleIds &&doc.exampleIds.length)}
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
  )
}

const mapDispatchToProps = dispatch => ({
  addStarred: (name, path, pageType) => () => dispatch(addStarred(name, path, pageType)),
  removeStarred: (path) => () => dispatch(removeStarred(path))
})

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  starred: state.starred,
  location: state.router.location,
  loading: state.loading,
  loggedIn: state.account.status === "logged-in",
})

export default connect(mapStateToProps, mapDispatchToProps)(FunctionPage)
