import React from 'react'
import { connect } from 'react-redux'

import { DocsItem, HowToItem } from '../util/listItems'

import { starredClicked, clearStarred } from '../../../../../../redux/actions/starred'
import { clearHistory } from '../../../../../../redux/actions/history'


//import './assets/nav.css'

const STARRED_COLLAPSED_LENGTH = 5
const HISTORY_COLLAPSED_LENGTH = 7
const MAX_HISTORY_LENGTH = 25

const StarredItem = ({type, link, text, code, onItemClick, title}) => {
  switch(type) {
    case 'identifier':
      return <DocsItem onItemClick={onItemClick} link={link} code={code} title={title}/>
    case 'howto':
      return <HowToItem onItemClick={onItemClick} link={link} text={text} code={code} title={title}/>
    default:
      return null
  }
}

const StarredItems = ({paths, starredMap, isExpanded, toggleHandler, showSizeToggles, onItemClick, clearHandler}) => {
  const showClear = paths && paths.length > 0
  return (
    <section>
      <h3 className={`starred${showSizeToggles ? ' clickable' : ''}${showClear ? ' with-button' : ''}`} onClick={showSizeToggles ? toggleHandler : undefined}>
        Starred
      </h3>
      { showClear && <button className='aside-toggle top' onClick={clearHandler}>clear</button> }
      <ul className='starred-items'>
        { paths &&
          paths.map((path, i) => {
            const item = starredMap[path]
            const text = item.pageType === 'howto' ? 'How to: ' : ''
            return <StarredItem title={item.name} onItemClick={onItemClick(item.path)} key={i} type={item.pageType} link={item.path} text={text} code={item.name}/>
          })
        }
      </ul>
      {showSizeToggles && <button onClick={toggleHandler} className='aside-toggle bottom'>
        {isExpanded ? 'collapse' : 'more'}
      </button>}
    </section>
  )
}

const HistoryItems = ({items, isExpanded, toggleHandler, showSizeToggles, clearHandler}) => {
  const showClear = items && items.length > 0
  return (
    <section>
      <h3 className={`${showSizeToggles ? 'clickable ' : ''}${showClear ? 'with-button' : ''}`} onClick={showSizeToggles ? toggleHandler : undefined}>
        History
      </h3>
      { showClear && <button className='aside-toggle top' onClick={clearHandler}>clear</button> }
      <ul className='history-items'>
        {
          items.map((item, i) => {
            const text = item.pageType === 'howto' ? 'How to: ' : '' 
            return <StarredItem title={item.name} key={i} type={item.pageType} link={item.path} text={text} code={item.name}/>
          })
        }
      </ul>
      {showSizeToggles && <button onClick={toggleHandler} className='aside-toggle bottom'>
        {isExpanded ? 'collapse' : 'more'}
      </button>}
    </section>
  )
}

class Nav extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      allowHistoryExpand: false,
      allowStarredExpand: false,
      starredExpanded: false,
      historyExpanded: false,
    }
  }

  starredToggleHandler = () => {
    this.setState({
      starredExpanded: !this.state.starredExpanded
    })
  }

  historyToggleHandler = () => {
    this.setState({
      historyExpanded: !this.state.historyExpanded
    })
  }

  static getDerivedStateFromProps(props, state) {
    const {history, starred} = props
    const allowHistoryExpand = history && history.previousPages && history.previousPages.length > HISTORY_COLLAPSED_LENGTH
      ? true
      : false
    const allowStarredExpand = starred && starred.starredPaths && starred.starredPaths.length > STARRED_COLLAPSED_LENGTH
      ? true
      : false
    if(allowHistoryExpand !== state.allowHistoryExpand || allowStarredExpand !== state.allowStarredExpand){
      return {
        allowHistoryExpand,
        allowStarredExpand,
        starredExpanded: state.starredExpanded,
        historyExpanded: state.historyExpanded
      }
    }
    return null
  }

  render() {
    return (
      <aside>
        { this.props.starred && this.props.starred.starredPaths.length > 0 &&<StarredItems
          clearHandler={this.props.clearStarred}
          toggleHandler={this.starredToggleHandler}
          isExpanded={this.state.starredExpanded} 
          showSizeToggles={this.state.allowStarredExpand}
          paths={ this.state.starredExpanded
            ? this.props.starred.starredPaths
            : this.props.starred.starredPaths.slice(0, STARRED_COLLAPSED_LENGTH)
          } 
          starredMap={this.props.starred.starredMap}
          onItemClick={this.props.starredClicked}/> }
        { (!this.props.starred || !this.props.starred.starredPaths || this.props.starred.starredPaths.length === 0) &&
          <section>
            <h3 className='starred'>
              Starred
            </h3>
            <div className='empty-starred'>
              Kite Doc pages that you star will show up here <br/><br/>Try it out!
            </div>
          </section>
        }
        { this.props.history && this.props.history.previousPages.length > 0 && <HistoryItems
          clearHandler={this.props.clearHistory}
          isExpanded={this.state.historyExpanded}
          toggleHandler={this.historyToggleHandler}
          showSizeToggles={this.state.allowHistoryExpand}
          items={ this.state.historyExpanded
            ? this.props.history.previousPages.slice(0, MAX_HISTORY_LENGTH)
            : this.props.history.previousPages.slice(0, HISTORY_COLLAPSED_LENGTH)
          }/> }
          { (!this.props.history || !this.props.history.previousPages || this.props.history.previousPages.length === 0) &&
          <section>
            <h3>
              History
            </h3>
            <div className='empty-history'>
              Kite Doc pages you visit will be saved here
            </div>
          </section>
        }
      </aside>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  history: state.history,
  starred: state.starred
})

const mapDispatchToProps = dispatch => ({
  starredClicked: (path) => () => dispatch(starredClicked(path)),
  clearHistory: () => dispatch(clearHistory()),
  clearStarred: () => dispatch(clearStarred()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Nav);
