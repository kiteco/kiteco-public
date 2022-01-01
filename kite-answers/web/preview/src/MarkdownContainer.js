import React from "react";

const REFRESH_TIMEOUT_MS = 3000;

class MarkdownContainer extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      value: "Paste Markdown Here!"
    };
    this.handleChange = this.handleChange.bind(this);
    this.handleSubmit = this.handleSubmit.bind(this);
    this.timeout = null;
  }

  handleChange(event) {
    this.setState({ value: event.target.value });
    clearTimeout(this.timeout);
    this.timeout = setTimeout(() => {
      this.props.renderAnswersPage(this.state.value);
    }, REFRESH_TIMEOUT_MS);
  }

  handleSubmit(event) {
    event.preventDefault();
    this.props.renderAnswersPage(this.state.value);
  }

  render() {
    return (
      <div className="markdownContainer">
        <form onSubmit={this.handleSubmit}>
          <textarea value={this.state.value} onChange={this.handleChange} />
          <br />
          <input type="submit" value="Render" />
        </form>
      </div>
    );
  }
}

export default MarkdownContainer;
