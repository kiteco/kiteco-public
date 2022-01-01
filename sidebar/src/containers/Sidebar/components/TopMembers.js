import React from 'react'

import { DocsToolTipTrigger } from './ToolTip'

//Max returned by symbol report api
const MAX_MEMBERS_HIDDEN = 5

//Originally, this component was written in functional, stateless style
//However, due to the need to fetch/show/hide members, it was converted to
//be stateful.
//Alternatively, could have decomposed this into a stateful wrapper with
//two functional stateless children (e.g. TopMembersExpanded, TopMembersPreview),
//but this appeared to be the cleanest conceptual model
class TopMembers extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      showAllMembers: false,
      displayMembers: this.props.members.slice(0, MAX_MEMBERS_HIDDEN),
    }
  }

  componentDidUpdate(prevProps, prevState) {
    if(prevProps.full_name !== this.props.full_name) {
      //then within kind switch
      this.setState({
        displayMembers: this.props.members.slice(0, MAX_MEMBERS_HIDDEN),
        showAllMembers: false,
      })
    }
    if(!prevProps.membersHaveLoaded && this.props.membersHaveLoaded) {
      this.setState({
        showAllMembers: true,
        displayMembers: this.props.members.slice()
      })
    }
  }

  toggleMembers = () => {
    //if currently showAllMembers, then about to move into hidden again
    let maxIndex = this.state.showAllMembers ? MAX_MEMBERS_HIDDEN : this.props.members.length
    const displayMembers = this.props.members.slice(0, maxIndex)
    this.setState({
      showAllMembers: !this.state.showAllMembers,
      displayMembers,
    })
  }

  moreMembersHandler = () => {
    if(this.props.membersHaveLoaded) {
      this.toggleMembers()
    } else {
      this.props.moreMembersHandler()
    }
  }

  getText = () => {
    if(!this.props.membersHaveLoaded || !this.state.showAllMembers) {
      return `See ${ this.props.total_members - this.state.displayMembers.length } more members in this ${this.props.kind}`
    } else {
      return '...hide'
    }
  }

  render() {
    return (
      <div className={`${this.props.className} docs__top-members`}>
        <h2>Popular members</h2>
        <ul className="columns">
          {this.state.displayMembers.map((member, index) => {
            let id = member.id 
            if(!id) {
              id = member.value
                ? member.value[0].id
                : ""
            }
            return <li
              key={"member-" + index + this.props.full_name}
            >
              <div className="member-name">
                <DocsToolTipTrigger
                  identifier={id}
                  language={this.props.language}
                >
                  <code>{member.name}</code>
                </DocsToolTipTrigger>
              </div>
              <div className="member-type">
                {member.value && [...new Set(member.value.map((value, index) => {
                    if(value.kind === 'instance') 
                      return value.type
                        ? value.type
                        : value.kind
                    return value.kind
                  }
                ))].join(' | ')}
              </div>
            </li>
          })}
        </ul>
        { this.props.total_members > MAX_MEMBERS_HIDDEN &&
          this.props.members &&
          <p className="docs__top-members__more" onClick={this.moreMembersHandler}>
            {this.getText()}
          </p>
        }
      </div>
    )
  }
}

export default TopMembers
