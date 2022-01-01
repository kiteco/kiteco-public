import React from 'react'
import { DocsToolTipTrigger } from './ToolTip'
import Navigation from '../../../components/Navigation'

import '../assets/title.css'

class Title extends React.Component {
  render() {
    if (this.props.parent && this.props.parent.id) {
      return (
        <div className="title__wrapper">
          <div className="label_wrapper">
            <h1>
              { <DocsToolTipTrigger
                  identifier={this.props.parent.id}
                  language={this.props.language}
                >
                  {this.props.parent.name}<wbr/>
                </DocsToolTipTrigger> }
              .
              { this.props.name }
              { this.props.status === "loading" &&
                <span className="spinner"></span>
              }
            </h1>
            {this.props.type &&
              <div className="title__type">
                {this.props.type}
                {this.props.type === 'instance' && this.props.typeId && this.props.typeRepr &&
                  <span className="title__type__id">&nbsp;of&nbsp;
                  <DocsToolTipTrigger
                    identifier={this.props.typeId}
                    language={this.props.language}
                  >
                    {this.props.typeRepr}<wbr/>
                  </DocsToolTipTrigger>
                  </span>
                }
              </div>
            }
          </div>
          <Navigation/>
        </div>
      )
    } else {
      return <div className="title__wrapper">
        <div className="label_wrapper">
          <h1>
            {this.props.full_name.split(".").map((crumb, i) => {
              if (i) {
                return <span key={i}><wbr/>.{crumb}</span>
              } else {
                return <span key={i}>{crumb}</span>
              }
            })}
            { this.props.status === "loading" &&
              <span className="spinner"></span>
            }
          </h1>
          {this.props.type &&
            <div className="title__type">{this.props.type}</div>
          }
        </div>
        <Navigation/>
      </div>
    }
  }
}

export default Title
