import React from 'react'
import { connect } from 'react-redux'
import { Helmet } from 'react-helmet'

import { BreadcrumbWrapper, Breadcrumbs, StarHeader, TypeIndicator } from '../../../util/Titles'
import Description from '../Description'
import HowToExamples from '../HowToExamples'

import Promo from '../Promo'

import { addStarred, removeStarred } from '../../../../../../../../redux/actions/starred'
import DocsSpinner from '../../../util/DocsSpinner'

import { fallback as meta } from '../../../util/meta-description'

const GenericPage = ({
  doc,
  name,
  type,
  language,
  starred,
  loading,
  location,
  addStarred,
  removeStarred,
  loggedIn,
}) => {
  const isStarred = starred.starredMap[location.pathname] ? true : false
  const clickHandler = isStarred ? removeStarred(location.pathname) : addStarred(name, location.pathname, 'identifier')
  return (
    <div className='items'>
      <Helmet>
        <title>{name} - {doc.ancestors.length > 0 ? `${doc.ancestors[0].name} - ` : ''}Python documentation - Kite</title>
        <meta
          name='description'
          content={meta({ doc, name})}
        />
      </Helmet>
      <div className='item'>
        {
          loading.isDocsLoading && <div className='intro'>
            <DocsSpinner />
          </div>
        }
        {
          !loading.isDocsLoading && <div className='intro'>
            <BreadcrumbWrapper>
              <Breadcrumbs items={doc.ancestors} />
              <StarHeader
                moduleName={name}
                type={type}
                isStarred={isStarred}
                clickHandler={clickHandler}
                path={location.pathname}
                position={doc.ancestors.length + 1}
              />
            </BreadcrumbWrapper>
            <TypeIndicator type={type} />
          </div>
        }
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
        </div>
      </div>
    </div>
  )
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  starred: state.starred,
  location: state.router.location,
  loading: state.loading,
  loggedIn: state.account.status === "logged-in",
})

const mapDispatchToProps = dispatch => ({
  addStarred: (name, path, pageType) => () => dispatch(addStarred(name, path, pageType)),
  removeStarred: (path) => () => dispatch(removeStarred(path))
})

export default connect(mapStateToProps, mapDispatchToProps)(GenericPage)
