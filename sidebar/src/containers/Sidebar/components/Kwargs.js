import React from 'react';

class Kwargs extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      expanded: false,
    }
  }

  componentDidUpdate(prevProps) {
    if (prevProps.full_name !== this.props.full_name) {
      this.setState({
        expanded: false,
      })
    }
  }

  toggleExpand = () => {
    this.setState({ expanded: !this.state.expanded })
  }

  render() {
    const { expanded } = this.state
    return <div className="docs__kwargs" id="kwargs">
      <h2>**{this.props.kwarg.name}</h2>
      <span className="docs__kwargs__expand" onClick={this.toggleExpand}>{ this.state.expanded ? 'Collapse' : 'Expand'}</span>
      {expanded && <ul>
        {this.props.kwargParameters && this.props.kwargParameters.map((kw, i) => {
        return <li key={"kw-" + kw.name} className={`docs__kwargs__row--even`}>
          <code className="docs__kwargs__name">{kw.name}</code>{kw.inferred_value_objs &&
            <code><span className="docs__kwargs__punc">: </span>{kw.inferred_value_objs.map((o, j) => {
              return <span key={"kw-" + o.token + j} className={`${o.type === 'val' ? 'docs__kwargs__value' : 'docs__kwargs__punc'}`}>{o.token}</span>
            })}</code>}
          </li>
      })}
      </ul>}
    </div>;
  }
}

export default Kwargs;