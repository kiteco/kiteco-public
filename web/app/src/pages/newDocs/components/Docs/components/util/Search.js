import React from 'react'
import { Link } from 'react-router-dom'
import onClickOutside from 'react-onclickoutside'

// helpers
import * as analytics from '../../../../../../utils/analytics'
import { queryCompletionPath } from '../../../../../../utils/urls'
import { transformQueryCompletions } from '../../../../../../utils/route-parsing'

// components
import PreloaderSpinner from '../../../../../../components/PreloaderSpinner'

const SearchResult = ({handleMouseEnter, handleClick, selected, completion}) => (
  <li
    onMouseEnter={handleMouseEnter}
    className={`
      code search__completion
      ${selected ? "search__completion--selected" : "search__completion--unselected"}
    `}
  >
    <Link
      className="search__completion__link"
      to={completion.path}
      onClick={e => handleClick(completion, e)}
    >
      <div className="search__completion__title">
        {completion.repr}
      </div>
      <div className="search__completion__detail">
        {completion.synopis}
      </div>
    </Link>
  </li>
)

class SearchResults extends React.Component {
  randomId = Math.random()
  scrollDisable = false

  handleClickOutside = evt => {
    this.props.handleOutsideClick()
  }

  // Function to handle mouse wheel
  handleMouseWheel = evt => {
    // add throttle to events
    if(!this.scrollDisable) {
      this.scrollDisable = true
      // run outside scroll event
      this.props.handleScrollEvent(evt)
      this.scrollDisable = false
    }
  }

  componentDidMount() {
    // Add mouseWheel event listener for Chrome/Safari/Edge/IE
    document.getElementById(`search-results${this.randomId}`)
      .addEventListener('mousewheel', this.handleMouseWheel)

    // Add mouseWheel event listener for Firefox
    document.getElementById(`search-results${this.randomId}`)
      .addEventListener("DOMMouseScroll", this.handleMouseWheel)
  }

  componentWillUnmount() {
    // Remove mouseWheel event listener for Chrome/Safari/Edge/IE
    document.getElementById(`search-results${this.randomId}`)
      .removeEventListener('mousewheel', this.handleMouseWheel)

    // Remove mouseWheel event listener for Firefox
    document.getElementById(`search-results${this.randomId}`)
      .removeEventListener("DOMMouseScroll", this.handleMouseWheel)
  }

  render() {
    const {
      completions,
      hidden,
      offset,
      isLoading,
      itemHeightSize,
      scrollOffset,
      visibleItems,
    } = this.props
    const listYPosition = itemHeightSize * scrollOffset

    return (
      <div
        id={`search-results${this.randomId}`}
        className={`search__completions${hidden ? '-hidden' : ''}`}
      >
        <ul
          className="search__completions-list"
          style={{transform: `translateY(${-listYPosition}px)`}}
        >
          {completions}
        </ul>
        {Array.isArray(completions) &&
          <div className="search-arrow-control">
            {isLoading ?
              <PreloaderSpinner
                containerClass="search-loader"
              />
              :
              <div className="search-arrow-control__wrapper code">
                {scrollOffset > 0 &&
                <div
                  className="search-arrow-control__item"
                >&#x2191;</div>
                }
                {(scrollOffset + visibleItems) < offset &&
                <div
                  className="search-arrow-control__item"
                >&#x2193;</div>
                }
              </div>
            }
          </div>
        }
      </div>
    )
  }
}

const HidingSearchResults = onClickOutside(SearchResults)

class Search extends React.Component {
  state = {
    query: "",
    querying: "",
    completions: [],
    hidden: true,
    isLoading: false,
    hasMoreItems: true,
    selected: 0,
    scrollOffset: 0,
    scrollMax: 0,
    offset: 0,
    limit: 20,
    visibleItems: 6,
    itemHeightSize: 35,
  }

  /**
   *  Function to move the highlight on one position up or down
   *  @param upward (boolean) - The highlight direction (true - up, false - down)
   */
  changeSelectedItem = upward => {
    const {
      visibleItems,
      scrollOffset,
      selected,
      offset,
      query,
      limit,
      isLoading,
      hasMoreItems
    } = this.state
    const currItemPosition = selected + 1

    // disable changes if data is fetching
    if (!isLoading) {
      if (upward && currItemPosition > 1) {

        this.setState(state => ({
          scrollOffset: scrollOffset < state.selected ? state.scrollOffset : state.scrollOffset - 1,
          selected: state.selected - 1,
        }))

      } else if (!upward && currItemPosition < offset) {

        const newItemPosition = currItemPosition + 1

        this.setState(state => ({
          scrollOffset: (scrollOffset + visibleItems) < newItemPosition ?
            state.scrollOffset + 1 : state.scrollOffset,
          selected: state.selected + 1,
        }))

        // fetch more data
        if (selected + visibleItems >= offset && hasMoreItems) {
          this.sendQuery({query, offset, limit})
        }
      }
    }
  }

  // Function to determine scroll direction and run external function with different props
  onScrollSearchResults = e => {
    // Determine the direction of the scroll (< 0 → up, > 0 → down).
    const delta = ((e.deltaY || -e.wheelDelta || e.detail) >> 10) || 1

    e.preventDefault()
    if (delta > 0) {
      this.changeSelectedItem()
    } else {
      this.changeSelectedItem(true)
    }
  }

  //keyboard navigations
  navigate = (e) => {
    const { noRedirect } = this.props
    switch(e.keyCode) {
      case 27: // esc key
        e.preventDefault()
        if (this.props.esc) {
          this.props.esc()
        }
        break
      case 38: { // up key
        if (this.state.selected > 0) {
          e.preventDefault()
          this.changeSelectedItem(true)
        }
        break
      }
      case 40: { // down key
        if (this.state.selected < this.state.completions.length - 1) {
          e.preventDefault()
          this.changeSelectedItem()
        }
        break
      }
      case 13: // enter key
        if (this.state.completions.length > 0) {
          const selected = this.state.completions[this.state.selected]
          if (noRedirect) {
            this.saveInQuery(selected)
          } else {
            this.props.push(selected.path)
            this.clearCompletions(selected.repr)
          }
        }
        break
      default:
        return
    }
  }

  sendQuery = ({query, offset, limit}) => {
    // lock down sending queries to the server
    this.setState({
      querying: query,
      hidden: false,
      isLoading: true,
    })

    this.props.get({
      url: queryCompletionPath({query, offset, limit}),
      skipStore: true,
    })
      .then(({ success, data }) => {
        if (this.state.query !== query && this.state.query !== "") {
          this.sendQuery({query: this.state.query, limit: this.state.limit})
          return false
        }
        if (success && data) {
          if (data.results) {
            const completions = transformQueryCompletions(data)
            this.setState(state => ({
              completions: [ ...completions, ...state.completions ],
              querying: "",
              offset: data.end,
              isLoading: false,
              hasMoreItems: true,
            }))
          } else {
            this.setState({
              selected: 0,
              querying: "",
              isLoading: false,
              hasMoreItems: false,
            })
          }
        } else {
          this.setState({
            completions: [],
            selected: 0,
            querying: "",
            offset: 0,
            isLoading: false,
          })
        }
      })
      .catch(error => {
        console.error(error)
        this.setState({
          completions: [],
          selected: 0,
          querying: "",
          offset: 0,
          isLoading: false,
        })
      })
  }

  //handles the input field value
  // and sends off queries at the appropriate time
  handleQueryChange = (event) => {
    const query = event.target.value

    // add analytics track on change in input field
    analytics.track('search_query_updated')

    if (query === "") {
      this.setState({
        query: query,
        completions: [],
        selected: 0,
        offset: 0,
        isLoading: false,
        scrollOffset: 0,
        scrollMax: 0,
        hasMoreItems: false,
      })
      if (this.props.noRedirect && typeof this.props.handleEmptyInput === 'function') {
        this.props.handleEmptyInput()
      }
    } else {
      this.setState({
        query: query,
        completions: [],
        scrollOffset: 0,
        scrollMax: 0,
        selected:0,
      })
    }
    if (!this.state.querying && query !== "") {
      this.sendQuery({ query, limit: this.state.limit })
    }
  }

  setSelected = (i) => {
    this.setState({
      selected: i,
    })
  }

  clearCompletions = val => {
    // add analytics track on submit input field value
    analytics.track('search_query_selected', {
      identifier: val.repr
    })

    this.setState({
      query: "",
      querying: "",
      completions: [],
      selected:0,
      hidden: true,
    })
  }

  saveInQuery = (val, e) => {
    if (e && typeof e.preventDefault === 'function') {
      e.preventDefault()
    }

    this.setState({
      query: val.repr,
      querying: "",
      completions: [],
      selected:0,
      hidden: true,
    })

    if (this.props.onSetQuery) {
      this.props.onSetQuery(val)
    }
  }

  handleClickOutsideSearch = () => {
    this.setState({ hidden: true })
  }

  handleSearchInputClick = () => {
    this.setState({ hidden: false })
  }

  handleSearchInputFocus = e => {
    const { query, isLoading, hidden } = this.state

    if (query.length > 0 && !isLoading && hidden) {
      this.handleQueryChange(e)
    }
  }

  render() {
    const { noRedirect } = this.props
    //need to transform path now
    let searchCompletions
    if (!this.state.querying && this.state.completions.length === 0 && this.state.query !== "") {
      searchCompletions = (
        <li className="search__completion search__no-completions">
          No results found for "<span className="code">{this.state.query}</span>"<br/>
          <span className="disclaimer">Try searching for an identifier starting with its package name</span>
        </li>
      )
    } else {
      searchCompletions = this.state.completions.map((completion, i) =>
        (
          <SearchResult
            handleMouseEnter={this.setSelected.bind(this, i)}
            handleClick={noRedirect ? this.saveInQuery : this.clearCompletions}
            key={`${completion.id}-${i}`}
            selected={this.state.selected === i}
            completion={completion}
          />
        )
      )
    }

    return (
      <div className='search'>
        <div className='search-wrapper'>
          <input className='code'
            ref={this.props.inputRef}
            type='text'
            placeholder={this.props.placeholder ? this.props.placeholder : 'Search Python Docs...'}
            value={this.state.query}
            onChange={this.handleQueryChange}
            onKeyDown={this.navigate}
            onClick={this.handleSearchInputClick}
            onFocus={noRedirect ? this.handleSearchInputFocus : undefined}
          />
          <HidingSearchResults
            hidden={this.state.hidden || !this.state.query}
            completions={searchCompletions}
            disableOnClickOutside={this.state.hidden}
            handleOutsideClick={this.handleClickOutsideSearch}
            offset={this.state.offset}
            isLoading={this.state.isLoading}
            itemHeightSize={this.state.itemHeightSize}
            scrollOffset={this.state.scrollOffset}
            visibleItems={this.state.visibleItems}
            handleScrollEvent={this.onScrollSearchResults}
            selected={this.state.selected}
          />
        </div>
      </div>
    )
  }
}

export default Search
