import React from 'react'

import './style.css'

class Select extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      selected: null,
      open: false,
    }
  }

  componentDidMount = () => {
    const { query } = this.props
    query(this.query)
  }

  query = () => {
    const { selected } = this.state
    return selected ? selected.id : ""
  }

  toggle = () => {
    if (!this.state.open) {
      document.addEventListener('click', this.clickOutside, false)
    } else {
      document.removeEventListener('click', this.clickOutside, false)
    }
    this.setState(state => ({
      ...state,
      open: !state.open,
    }))
  }

  select = selection => () => {
    this.setState( state => ({
      ...state,
      selected: selection,
    }))
    this.toggle()
  }

  clickOutside = (e) => {
    if (!this.node.contains(e.target)) {
      this.toggle()
    }
  }

  render() {
    const { selected, open } = this.state
    const { options } = this.props
    return <div className="select" ref={node => this.node = node}>
      <div className="select__toggle" onClick={this.toggle}>
        { selected === null
          ? "Choose…"
          : selected.content
        }
      </div>
      { open &&
        <div className="select__options">
          { options.map(o => <Option
            key={o.id}
            {...o}
            select={this.select(o)}
          />) }
        </div>
      }
    </div>
  }
}

const Option = ({ id, content, select=(() => {}) }) =>
  <div className="select__option" onClick={select} >
    { content }
  </div>

export default Select
