import React from 'react'
import { connect } from 'react-redux'
import { Helmet } from 'react-helmet'

import { Link } from 'react-router-dom'
import { BreadcrumbWrapper, Breadcrumbs, StarHeader } from '../../../util/Titles'
import { fallback as meta } from '../../../util/meta-description'
import Description from '../Description'
import Promo from '../Promo'
import HowToExamples from '../HowToExamples'

import { addStarred, removeStarred } from '../../../../../../../../redux/actions/starred'
import DocsSpinner from '../../../util/DocsSpinner'

const InstanceIndicator = ({types}) => {
  return (
    <div className='instance-indicator'>
      instance of {types.map((type, i) =>
        <span key={i} className="code">
          <Link to={type.path}>
            {type.name}
          </Link>
          {i < types.length - 1 && <span className='punctuation'> &#10093; </span>}
        </span>
      )}
    </div>
  )
}

const InstancePage = ({
  doc,
  name,
  type,
  language,
  loggedIn,
  location,
  starred,
  addStarred,
  removeStarred,
  loading }) => {
  const isStarred = starred.starredMap[location.pathname] ? true : false
  const clickHandler = isStarred ? removeStarred(location.pathname) : addStarred(name, location.pathname, 'identifier')

  return <div className='items'>
    <Helmet>
      <title>{name} - {doc.ancestors.length > 0 ? `${doc.ancestors[0].name} - ` : ''}Python documentation - Kite</title>
      <meta name="description" content={meta({ doc, name })}/>
    </Helmet>
    <div className='item'>
      { loading.isDocsLoading && <div className='intro'>
        <DocsSpinner />
      </div> }
      { !loading.isDocsLoading && <div className='intro'>
        <BreadcrumbWrapper>
          <Breadcrumbs items={doc.ancestors}/>
          <StarHeader
            moduleName={name}
            isStarred={isStarred}
            clickHandler={clickHandler}
            type='instance'
            path={location.pathname}
            position={doc.ancestors.length + 1}
          />
        </BreadcrumbWrapper>
        {doc.types && doc.types.length > 0 && <InstanceIndicator types={doc.types} />}
      </div> }
      <div className='docs'>
        { (doc.description_html || doc.documentation_str) &&
          <Description
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
}

const mapStateToProps = (state, ownProps) => {
  return {
    ...ownProps,
    location: state.router.location,
    starred: state.starred,
    loading: state.loading
  }
}

const mapDispatchToProps = dispatch => ({
  addStarred: (name, path, pageType) => () => dispatch(addStarred(name, path, pageType)),
  removeStarred: (path) => () => dispatch(removeStarred(path))
})

export default connect(mapStateToProps, mapDispatchToProps)(InstancePage)
