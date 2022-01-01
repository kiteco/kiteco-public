import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import { Shortcuts } from 'react-shortcuts'
import { push } from 'react-router-redux'

import { GET } from '../../actions/fetch'
import { forceCheckOnline } from '../../actions/system'

import './assets/search.css'
import { metrics } from '../../utils/metrics'
import { searchQueryCompletionPath } from '../../utils/urls'


import AutosearchToggle from './AutosearchToggle'

const SearchResult = ({handleMouseEnter, selected, completion, onClick}) => (
  <li
    onMouseEnter={handleMouseEnter}
    onClick={onClick}
    className={`
      search__completion
      ${selected ? "search__completion--selected" : ""}
    `}
  >
    <Link
      className="search__completion__link"
      to={completion.path}
      onClick={onClick}
    >
      <div className="search__completion__title">
        {completion.repr}
      </div>
      <div className="search__completion__detail">
        {completion.documentation}
      </div>
    </Link>
  </li>
)

class Search extends React.Component {
    constructor(props) {
      super(props)
      this.state = {
        query: "",
        querying: "",
        completions: [],
        selected:0,
        autosearchHintEnabled: true,
        dataUnavailable: false,
        isFocused: false,
      }
    }

    componentDidMount() {
      // Hacky solution to detect clicks outside of search box
      document.addEventListener('click', this.handleClick, false)
    }

    componentWillUnmount() {
      // Hacky solution to detect clicks outside of search box
      document.removeEventListener('click', this.handleClick, false)
    }

    // Hacky solution to detect clicks outside of search box
    handleClick = (e) => {
      if (!ReactDOM.findDOMNode(this).contains(e.target)) {
        this.escape()
      }
    }

    //keyboard navigations
    navigate = (action, e) => {
      switch(action) {
        case 'ESC': // esc key
          this.escape();
          break
        case 'MOVE_UP': // up key
          this.moveUp();
          break
        case 'MOVE_DOWN': // down key
          this.moveDown();
          break
        case 'TAB':
          e.preventDefault()
          this.tab()
          break
        case 'CONFIRM': // enter key
          this.confirm();
          break
        default:
          return
      }
    }

    confirm() {
      if (this.state.completions.length > 0) {
        this.handleResultSelected()
        const selected = this.state.completions[this.state.selected]
        this.props.push(selected.path)
        this.clearSearch()
      }
    }

    moveUp() {
      if (this.state.selected > 0) {
        this.setState({
          selected: this.state.selected - 1,
        })
      }
    }

    moveDown() {
      if (this.state.selected < this.state.completions.length - 1) {
        this.setState({
          selected: this.state.selected + 1,
        })
      }
    }

    tab() {
      const selected = this.state.completions[this.state.selected]
      if(selected && selected.repr !== this.state.query) {
        this.setState({
          query: selected.repr
        })
        if (!this.state.querying && selected.repr !== "") {
          this.sendQuery(selected.repr)
        }
      }
    }

    escape() {
      if (this.props.esc) {
        this.props.esc()
      } else {
        this.clearSearch();
      }
    }

    //send off input query for completions
    sendQuery = (query) => {
      // lock down sending queries to the server
      this.setState({
        querying: query,
      })

      this.props.get({
        url: searchQueryCompletionPath({query}),
        skipStore: true,
      })
        .then(({ success, data, status }) => {
          if (success && data) {
            if(data.data_unavailable) {
              this.setState({
                completions: [],
                selected: 0,
                querying: "",
                dataUnavailable: true,
              })
            } else {
              data.results.forEach(completion => {
                completion.result.path = `/docs/${completion.result.id}`
              })
              this.setState({
                completions: data.results.map(res => res.result),
                selected: 0,
                querying: "",
                dataUnavailable: false,
              })
            }
          } else {
            // response from kited indicates search isn't
            // implemented in this context
            // a bad gateway indicates that kited proxied the request
            // but it couldn't go through, which we'll expose similarly to
            // a user
            if(status && (status === 501 || status === 502)) {
              this.setState({
                completions: [],
                selected: 0,
                querying: "",
                dataUnavailable: true,
              })
            } else {
              this.setState({
                completions: [],
                selected: 0,
                querying: "",
                dataUnavailable: false,
              })
            }
          }
        })
        .then(() => {
          // if the query changed since the most recent call
          // send the most recent query
          if (this.state.query !== query && this.state.query !== "") {
            this.sendQuery(this.state.query)
          }
        })
        .catch(error => {
          console.error(error)
          this.setState({
            completions: [],
            selected: 0,
            querying: "",
            dataUnavailable: false,
          })
        })
    }

    //handles the input field value
    // and sends off queries at the appropriate time
    handleQueryChange = (event) => {
      const query = event.target.value
      if (query === "") {
        this.setState({
          query: query,
          completions: [],
          selected: 0,
          dataUnavailable: false
        })
      } else {
        if (this.state.query === "") {
          metrics.incrementCounter('sidebar_search_query_started')
        }
        this.setState({
          query: query,
        })
      }
      if (!this.state.querying && query !== "") {
        this.sendQuery(query)
      }
    }

    //set which result is currently selected through keyboard
    // or mouse
    setSelected = (i) => {
      this.setState({
        selected: i,
      })
    }

    handleSearchResultClick = () => {
      this.handleResultSelected()
      this.clearSearch()
    }

    handleResultSelected = () => {
      metrics.incrementCounter('sidebar_search_result_selected')
    }

    clearSearch = () => {
      this.setState((prevState, props) => {
        return {
          ...prevState,
          completions: [],
          selected: 0,
          querying: '',
          query: '',
          dataUnavailable: false,
        }
      });
    }

    gateSearchbar = type => () => {
      if(!this.state.isFocused && type === 'focus') {
        this.setState({ isFocused: true })
      }
      this.props.forceCheckOnline().then(({ success, isOnline }) => {
        if(success && isOnline) {
          if(this.state.dataUnavailable) {
            this.setState({ dataUnavailable: false })
          }
        } else {
          if(!this.state.dataUnavailable) {
            this.setState({ dataUnavailable: true })
          }
        }
      })
    }

    onBlur = () => {
      if(this.state.isFocused) {
        this.setState({ isFocused: false })
      }
    }

    render() {
      const {
        className,
      } = this.props

      const {
        querying,
        completions,
        query,
        selected,
        dataUnavailable,
        isFocused,
      } = this.state
      let searchCompletions
      if(!querying && dataUnavailable && isFocused) {
        searchCompletions = (
          <div className="search__no-completions">
            You have to be online to use search
          </div>
        )
      } else if (!querying && completions.length === 0 && query !== "") {
        searchCompletions = (
          <div className="search__no-completions">
            No results found for "{query}". Try searching for an identifier starting with its package name.
          </div>
        )
      } else {
        searchCompletions = completions.map((completion, i) =>
          (
            <SearchResult
              handleMouseEnter={this.setSelected.bind(this, i)}
              key={completion.id}
              onClick={this.handleSearchResultClick}
              selected={selected === i}
              completion={completion}
            />
          )
        )
      }

      return (
        <Shortcuts
          name='Search'
          alwaysFireHandler={true}
          handler={this.navigate}
          className="search__form__shortcut"
        >
        <div className={`search ${className || ""}`}>
          <div className="search__form">
            <div className="search__form__wrapper">
                <input
                  className="search__input"
                  ref={ el => this.searchInputEl = el }
                  type="text"
                  placeholder="Search…"
                  value={ query }
                  onChange={ this.handleQueryChange }
                  onFocus={ this.gateSearchbar('focus') }
                  onBlur={ this.onBlur }
                  onClick={ this.gateSearchbar('click') }
                />
                <div className="search-icon"></div>
            </div>
            <AutosearchToggle/>
          </div>
          <ul className="search__completions">
            {searchCompletions}
            {searchCompletions.length !== 0 && <li className="search__completions__info">↑&nbsp;↓&nbsp;Tab&nbsp;Esc</li>}
          </ul>
        </div>
      </Shortcuts>
      )
  }
}

const mapDispatchToProps = dispatch => ({
  get: params => dispatch(GET(params)),
  push: params => dispatch(push(params)),
  forceCheckOnline: () => dispatch(forceCheckOnline()),
})

const mapStateToProps = (state, ownProps) => ({
})

export default connect(mapStateToProps, mapDispatchToProps)(Search)
