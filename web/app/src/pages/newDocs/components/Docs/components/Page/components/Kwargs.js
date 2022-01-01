import React from 'react'

const Kwarg = ({ kw }) => {
  return (
    <div className='kwarg'>
      <pre>
        <code className='with-syntax-highlighting code'>
          <span>{kw.name}</span>
          {kw.types && kw.types.length && <span className='punctuation'>: </span>}
          {kw.types && kw.types.length && kw.types.map((type, i) => {
            return <span key={i}>
              <span className='keyword'>{type.repr}</span>
              {i < kw.types.length - 1 && <span className='punctuation small-text'> &#10072; </span>}
            </span>
          })}
        </code>
      </pre>
    </div>
  )
}

class Kwargs extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      collapsed: true,
      kwargsToShow: props.kwargs ? props.kwargs.slice(0, 5) : []
    }
  }

  toggleExpand = () => {
    this.setState({
      collapsed: !this.state.collapsed,
      kwargsToShow: this.state.collapsed ? this.props.kwargs : this.props.kwargs.slice(0,5)
    })
  }

  render() {
    const {kwargsToShow, collapsed} = this.state
    return (
      <section className='kwargs'>
      <h3>
        **KW
      </h3>
      <div>
        {kwargsToShow.map((kw, i) =>
          <Kwarg key={i} kw={kw} />
        )}
        {kwargsToShow.length > 5 &&
          <button className='expand-kwargs' onClick={this.toggleExpand}>{collapsed
            ? 'show more'
            : 'collapse'
          }</button>
        }
      </div>
    </section>
    )
  }
}

export default Kwargs
