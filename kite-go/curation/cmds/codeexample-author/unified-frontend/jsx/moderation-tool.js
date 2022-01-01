React = require('react/addons');
$ = require('jquery');

require('./global-utils.js');

CodeExampleEditor = require('./code-example-editor.js');
CheckboxFilter = require('./checkbox-filter.js');
EditorModes = require('./example-editor-modes.js');

var ExampleStates = ['in_progress', 'pending_review', 'needs_attention', 'approved'];

var ModerationTool = React.createClass({
  getInitialState: function() {
    return {
      loading: true,
      examples: [],
      statusFilter: ['pending_review'],
    };
  },
  fetchExamples: function(statusFilter) {
    var lang = 'python';
    var pkg = 'json';
    $.ajax({
      type: 'GET',
      url: '/api/examples/query',
      data: {
        'statuses': statusFilter.join(','),
      },
      dataType: 'json',
      success: (data) => {
        this.setState({
          loading: false,
          pkg: pkg,
          examples: data,
        });
        console.log(data);
      },
      error: (data) => {
        console.log(arguments);
      },
    });
  },
  componentDidMount: function() {
    this.fetchExamples(this.state.statusFilter);
  },
  changeFilter: function(newFilter) {
    this.setState({
      filter: newFilter,
      loading: true,
    });
    this.fetchExamples(newFilter);
  },
  render: function() {
    return <div className="moderation-tool">
      <header>
        <h3>Curated snippets - moderation view</h3>
        <CheckboxFilter
          onFilterChanged={this.changeFilter}
          options={ExampleStates}
          initialSelection={this.state.statusFilter} />
      </header>
      {map(this.state.examples, example =>
        <CodeExampleEditor
          key={example.backendId}
          lang={example.language}
          pkg={example.package}
          initExample={example}
          mode={EditorModes.MODERATE}
          scrollElt={window} />
      )}
    </div>;
  },
});

module.exports = ModerationTool;
