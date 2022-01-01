import React from 'react'
import { connect } from 'react-redux'

import * as styleActions from '../../../../../../../../redux/actions/style'
import { addStarred, removeStarred } from '../../../../../../../../redux/actions/starred'
import { sortMembers } from '../../../../../../../../redux/actions/docs'

import { DocsItem } from '../../../util/listItems'
import { BreadcrumbWrapper, Breadcrumbs, StarHeader, TypeIndicator } from '../../../util/Titles'
import DocsSpinner from '../../../util/DocsSpinner'

//import './assets/module-nav.css'

//Assume presorted along popularity dimension
const getMemberNameArr = (name) => {
  return name.split(/(?=[A-Z_])/)
}

// defaults are var(--text-subtle) and var(--background) from light-mode CSS
const popSVG = (className='', color='hsl(210, 20%, 60%)', background='white') => (
  <svg
    className={className} 
    width='13' 
    height='12' 
    viewBox='0 0 13 12' 
    version='1.1' 
    xmlns='http://www.w3.org/2000/svg' 
    xmlnsXlink='http://www.w3.org/1999/xlink'>
    <g id='Canvas' fill='none'>
    <g id='Frame 2' clipPath='url(#clip0)'>
    <g id='Popularity'>
      <rect id='Line' y='1' x='9' width='2' height='12' fill={color} style={{stroke: 'none'}}/>
      <rect id='Cover' y='0' x='8' width='4' height='2' fill={background} style={{stroke: 'none'}}/>
      <rect id='Line' y='1' x='5' width='2' height='12' fill={color} style={{stroke: 'none'}}/>
      <rect id='Cover' y='0' x='4' width='4' height='6' fill={background} style={{stroke: 'none'}}/>
      <rect id='Line' y='1' x='1' width='2' height='12' fill={color} style={{stroke: 'none'}}/>
      <rect id='Cover' y='0' x='0' width='4' height='10' fill={background} style={{stroke: 'none'}}/>
    </g>
    </g>
    </g>
      <defs>
        <clipPath id="clip0">
        <rect width="13" height="12" fill={background}/>
        </clipPath>
      </defs>
  </svg>
)

const MembersList = ({ members, sortMembers, memberSortCriteria }) => {
  return (
    <section>
      <h3>
        Members
        <span className="sort-container">
          <span 
            className={`sort-button icon`} 
            onClick={sortMembers('popularity')}>{popSVG(memberSortCriteria === 'popularity' ? 'highlight' : '')}</span>
          <span 
            className={`sort-button ${memberSortCriteria === 'name' ? 'highlight' : ''}`} 
            onClick={sortMembers('name')}>Aa</span>
        </span>
      </h3>
      <div>
        <ul className='code with-popularity'>
          {members.map((member, i) => {
            const name = getMemberNameArr(member.name)
            return <DocsItem
              key={i}
              popularity={member.popularity}
              link={member.link}
              code={name}
              showType={true}
              type={member.type}/>
        })}
        </ul>
      </div>
    </section>
  )
}

class ModuleNav extends React.Component {

  constructor(props) {
    super(props)
    this.breadcrumbs = React.createRef()
    this.state = {
      isStarred: false,
      clickHandler: null,
      currentPath: "",
      currentName: ""
    }
  }

  componentDidUpdate() {
    if(this.breadcrumbs.current) {
      const breadcrumbsHeight = this.breadcrumbs.current.clientHeight
      if(breadcrumbsHeight !== this.props.style.extraIntroHeight) {
        this.props.setExtraIntroHeight(breadcrumbsHeight)
      }
    }
  }

  static getDerivedStateFromProps(props, state) {
    const { location={}, name, starred } = props
    const isStarred = starred && starred.starredMap[location.pathname] ? true : false
    if(starred && (isStarred !== state.isStarred || !state.clickHandler || location.pathname !== state.currentPath || name !== state.currentName)) {
      return {
        isStarred,
        currentName: name,
        currentPath: location.pathname,
        clickHandler: isStarred
          ? props.removeStarred(location.pathname)
          : props.addStarred(name, location.pathname, 'identifier')
      }
    }
    return null
  }

  render() {
    const {
      name,
      ancestors,
      members,
      style,
      type,
      loading,
      location={},
      sortMembers,
      memberSortCriteria,
    } = this.props
    return (
      <nav className='nav'>
        { loading.isDocsLoading && <div className={`intro fixed-height-intro ${style.extraIntroClass}`}>
          <DocsSpinner />
        </div> }
        { !loading.isDocsLoading && <div className={`intro fixed-height-intro ${style.extraIntroClass}`}>
          <BreadcrumbWrapper>
            <Breadcrumbs refFromParent={this.breadcrumbs} items={ancestors} />
            <StarHeader
              type={type}
              moduleName={name}
              isStarred={this.state.isStarred}
              clickHandler={this.state.clickHandler}
              path={location.pathname}
              position={ancestors.length + 1}
            />
          </BreadcrumbWrapper>
          <TypeIndicator type={type} />
        </div> }
        <div className='index'>
          {members && members.length > 0 && <MembersList 
            members={members}
            memberSortCriteria={memberSortCriteria} 
            sortMembers={sortMembers}/>}
        </div>
      </nav>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  style: state.style,
  starred: state.starred,
  location: state.router.location,
  loading: state.loading,
  memberSortCriteria: state.docs.memberSortCriteria,
})

const mapDispatchToProps = dispatch => ({
  setExtraIntroHeight: (height) => dispatch(styleActions.setExtraIntroHeight(height)),
  addStarred: (name, path, pageType) => () => dispatch(addStarred(name, path, pageType)),
  removeStarred: (path) => () => dispatch(removeStarred(path)),
  sortMembers: (criteria) => () => dispatch(sortMembers(criteria)),
})

export default connect(mapStateToProps, mapDispatchToProps)(ModuleNav)
