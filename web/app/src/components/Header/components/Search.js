import React from 'react'
import ReactDOM from 'react-dom'

// import SearchCore from '../../../components/Search'

import '../assets/search.css'

class Search extends React.Component {
    constructor(props) {
      super(props)

      this.state = {
        open: false,
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
        this.setState({
          open: false,
        })
      }
    }

    // toggle search modal visibility
    toggleShow = () => {
      this.setState({
        open: !this.state.open,
      }, () => {
        if (this.state.open) {
          ReactDOM.findDOMNode(this.input).focus()
        }
      })
    }

    render() {
      const { light } = this.props
      return (
        <div className="header-search">
          <div
            className={`
              header-search__modal
              ${this.state.open ? "" : "header-search__modal--closed"}
            `}
          >
            {/* <SearchCore
              esc={this.toggleShow}
              inputRef={input => {this.input = input}}
              get={this.props.get}
              push={this.props.push}
            /> */}
          </div>
          <button onClick={this.toggleShow} className={`search-icon ${light ? "search-icon--light" : ""}`}>
          </button>
        </div>
      )
  }
}

export default Search
