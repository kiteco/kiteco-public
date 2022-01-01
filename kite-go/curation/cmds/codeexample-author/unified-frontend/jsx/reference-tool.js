React = require('react/addons');

require('./global-utils.js');

var Events = new Dispatcher();

function items(obj) {
  var r = [];
  for (var prop in obj) {
    if (obj.hasOwnProperty(prop)) {
      r.push([prop, obj[prop]]);
    }
  }
  return r;
}

var ReferenceModel = function() {
  var endpoint = function() {
    return INIT_DATA.codeExampleUrlBase + Array.prototype.slice.call(arguments).join('/');
  };
  var queryString = function(params) {
    var paramStrings = items(params).map((paramPair) => paramPair.join('='));
    return '?' + paramStrings.join('&');
  };
  var retrieveStackoverflowData = function(query) {
    $.ajax({
      url: endpoint('stackoverflow') + queryString({q: query}),
      type: 'GET',
    })
    .done(function(response) {
      Events.trigger('newStackoverflowData', response);
    })
    .fail(function() {
      console.log("error");
    });
  };

  var retrieveChildrenMenus = function(path) {
    $.ajax({
      url: endpoint('path', 'python', path),
      type: 'GET',
    })
    .done(function(response) {
      Events.trigger('newMenuItems-' + path , response);
    })
    .fail(function() {
      console.log("error");
    });
  };

  var retrieveSuggestions = function(path) {
    $.ajax({
      url: endpoint('path', 'python', path, 'suggestions'),
      type: 'GET',
    })
    .done(function(response) {
      Events.trigger('newSuggestions-' + path , response);
    })
    .fail(function() {
      console.log("error");
    });
  };

  var retrieveExamples = function(path) {
    $.ajax({
      url: endpoint('path', 'python', path, 'codeExamples'),
      type: 'GET',
    })
    .done(function(response) {
      Events.trigger('codeExamples-' + path , response);
    })
    .fail(function() {
      console.log("error");
    });
  };

  return {
    retrieveStackoverflowData : retrieveStackoverflowData,
    retrieveChildrenMenus : retrieveChildrenMenus,
    retrieveSuggestions : retrieveSuggestions,
    retrieveExamples : retrieveExamples,
  };
}();

var ReferenceTool = React.createClass({
  render: function() {
    return <div className="referenceContainer">
      <div className="referenceToolTitle">Reference</div>
      <StackoverflowViewer />
      <div className="scrollableContainer">
        <h2>Modules & functions in {this.props.pkg}</h2>
        <HierarchyLevel path={this.props.pkg} />
      </div>
    </div>;
  },
});

var StackoverflowViewer = React.createClass({
  handleClick: function() {
    this.setState({
      open: !this.state.open,
    });
  },
  handleSearchboxKeyDown: function(event) {
    if (event.keyCode === 13) {
      event.preventDefault();
      this.changeQuery(event.target.textContent);
    }
  },
  changeQuery: function(newQuery) {
    if (newQuery !== this.state.query) {
      this.setState({
        query: newQuery,
      });
    }
  },
  componentDidMount: function() {
    Events.register('newStackoverflowData', (newData) => {
      this.setState({
        posts: (newData === null) ? [] : newData,
        loadingResults: false,
      });
    });

    Events.register('newSuggestedSearch', (newQuery) => {
      this.setState({
        query: newQuery,
        open: true,
      });
    });

    Events.register('openStackoverflowviewer', (newQuery) => {
      this.handleClick();
    });
  },
  componentWillUpdate: function(nextProps, nextState) {
    if (this.state.query !== nextState.query) {
      this.setState({
        loadingResults: true,
      });
      ReferenceModel.retrieveStackoverflowData(nextState.query);
    }
  },
  getInitialState: function() {
    return {
      open: false,
      query: null,
      posts: [],
      loadingResults: false,
    };
  },
  handleClickQuestion: function() {
    $(event.target).parents('.stackoverflowResult').toggleClass('open');
  },
  render: function() {
    var spinner = this.state.loadingResults && <div className="spinner"></div>;
    var posts = this.state.posts.map((elem) =>
      <div className="stackoverflowResult">
        <div className="question" onClick={this.handleClickQuestion.bind(this)}>
          <div className="questionTitle">{elem.question.post.Title}</div>
          <div className="questionBody" dangerouslySetInnerHTML={{__html: elem.question.post.Body}} />
        </div>
        <div className="answers">
          {map(elem.answers, (elemAnswer) =>
            <div className="answer">
              <div className="answerTitle"></div>
              <div className="answerBody" dangerouslySetInnerHTML={{__html: elemAnswer.post.Body}}/>
            </div>
          )}
        </div>
      </div>);

    return <div className="stackoverflowViewer" data-open={this.state.open}>
      <div className="tab" onClick={this.handleClick}></div>
      <div className="stackoverflowContainer">
        <div className="heading">
          <h3>Stackoverflow</h3>
          <div className="searchbox" contentEditable="true" onKeyDown={this.handleSearchboxKeyDown}>
            {this.state.query}
          </div>
        </div>
        <div className="stackoverflowResults">
          {spinner}
          {posts}
        </div>
      </div>
    </div>;
  },
});

// TBD if this is going to actually be useful or not
var Breadcrumb = React.createClass({
  getInitialState: function() {
    return  {
      path: ['python'],
    };
  },
  render: function() {
    return <div className="breadcrumb">
      {this.state.path.map((pathSection) => <div className="section">{pathSection}</div>)}
    </div>;
  },
});

var HierarchyLevel = React.createClass({
  // API discussion is here: https://quip.com/PCVYAwV5qDTx
  // Returns response from backend based on code example service. JSON structure TBD.
  loadExamplesData: function() {
    if (this.state.path !== '') {
      Events.register('codeExamples-' + this.state.path, (newData) => {
        this.setState({
          examples: (newData.Signatures === null && newData.Cooccurrence === null) ? null : newData,
        });
      });

      ReferenceModel.retrieveExamples(this.state.path);
    }
  },
  // Returns array of children menu items. If this returns an empty array,
  // it means we're at a leaf node (in Python, this happens to be a function)
  loadChildrenPaths : function() {
    Events.register('newMenuItems-' + this.state.path, (newData) => {
      this.setState({
        menuItems: (newData.children === null) ? [] : newData.children,
        loadingMenuItems: false,
      });
    });

    this.setState({
      loadingMenuItems: true,
    });

    ReferenceModel.retrieveChildrenMenus(this.state.path);
  },
  // Returns array of suggested query objects. Each object has a query text and a state.
  loadSuggestions : function() {
    if (this.state.path !== '') {
      Events.register('newSuggestions-' + this.state.path , (newData) => {
        this.setState({
          suggestions: (newData.suggestions === null) ? [] : newData.suggestions,
        });
      });

      ReferenceModel.retrieveSuggestions(this.state.path);
    }
  },
  // Set the state of this hierarchy level
  loadDataForThisLevel: function(path) {
    this.loadExamplesData();
    this.loadSuggestions();
    this.loadChildrenPaths();
  },
  getInitialState: function() {
    return {
      path: this.props.path,
      suggestions: [],
      menuItems: [],
      examples: null,
      displaySuggestions: true,
      loadingMenuItems: false,
      loadingExamples: false,
      loadingSuggestions: false,
    };
  },
  handleClick: function(childComponent) {
    // TODO: improve this so that if we collapse child menus, the suggestions show up again
    if (childComponent.state.open === false) {
      this.setState({
        displaySuggestions: false,
      });
    }
  },
  componentDidMount: function() {
    this.loadDataForThisLevel();
  },
  render: function() {
    var itemsMenu = (this.props.hierarchyType !== 'method' ) && this.state.menuItems.map((menuItem) =>
        <MenuItem menuItemData={menuItem} onClick={this.handleClick} />);
    var loadingAny = this.state.loadingMenuItems || this.state.loadingExamples || this.state.loadingSuggestions;
    var spinner = loadingAny && <div className="spinner"></div>;
    var suggestions = this.state.displaySuggestions && <SuggestedQueries suggestions={this.state.suggestions}/>;
    var examples = this.state.examples && <ExampleViewer examples={this.state.examples}/>;
    return <ul className="hierarchyLevel">
      {spinner}
      {suggestions}
      {examples}
      {itemsMenu}
    </ul>;
  },
});

var MenuItem = React.createClass({
  getInitialState: function() {
    return {
      open: false,
    };
  },
  clickTitle: function() {
    this.setState({
      open: !this.state.open,
    });
    this.props.onClick(this);
  },
  render: function() {
    var nextHierarchyLevel = this.state.open &&
      <HierarchyLevel path={this.props.menuItemData.path} hierarchyType={this.props.menuItemData.type} />;

    var label = this.props.menuItemData.menuLabel;
    if (this.props.menuItemData.type === 'method') {
      label += '()';
    }

    return <li className="menuItem" data-open={this.state.open} data-type={this.props.menuItemData.type}>
      <div className="label" onClick={this.clickTitle}>
        <div className="title">
          <span className="disclosure-arrow">{this.state.open ? '▼' : '►'}</span>
          {label}
        </div>
        <div className="frequency">{asPercentage(this.props.menuItemData.freq)}</div>
      </div>
      {nextHierarchyLevel}
    </li>;
  },
});

function asPercentage(x) {
  return Math.floor(x*100) + '%';
}

var ExampleViewer = React.createClass({
  getInitialState: function() {
    return {
      signatures: this.props.examples.Signatures,
      cooccurrence: this.props.examples.Cooccurrence,
    };
  },
  handleClickAccordion : function(event) {
    $(event.target).parents('.exampleBucket').toggleClass('open');
  },
  render: function() {
    var signatures_col = map(this.state.signatures, (signature) =>
        <div className="exampleBucket">
          <div className="exampleTitle" onClick={this.handleClickAccordion}>
            <div className="titleLabel">{signature.Pattern.Signature}</div>
            <div className="frequency">{asPercentage(signature.Pattern.Frequency)}</div>
          </div>
          <div className="snippetContainer">
            {map(signature.Snippets, (snippet) =>
              <div className="snippet">
                <pre>
                  <code>
                    {snippet.Code}
                  </code>
                </pre>
              </div>)}
          </div>
        </div>);

    var cooccurrence_col = map(this.state.cooccurrence, (cooccurrence) =>
        <div className="exampleBucket">
          <div className="exampleTitle" onClick={this.handleClickAccordion}>
            <div className="titleLabel">{cooccurrence.Pattern.Pattern.join(', ')}</div>
            <div className="frequency">{asPercentage(cooccurrence.Pattern.Frequency)}</div>
          </div>
          <div className="snippetContainer">
            {map(cooccurrence.Snippets, (snippet) =>
              <div className="snippet">
                <pre>
                  <code>
                    {snippet.Code}
                  </code>
                </pre>
              </div>)}
          </div>
        </div>);

    return <div className="examplesContainer">
      {map(signatures_col, (rows) =>
        <div className="signaturesContainer">
          <h4>Common invocation patterns</h4>
          {rows}
        </div>
      )}
      {map(cooccurrence_col, (rows) =>
        <div className="cooccurrenceContainer">
          <h4>Cooccurrent functions</h4>
          {rows}
        </div>
      )}
    </div>;
  },
});

var SuggestedQueries = React.createClass({
  getInitialState: function() {
    return {
      suggestions: this.props.suggestions,
    };
  },
  handleClickSuggestion: function(event) {
    if ($(event.target).is('input') || $(event.target).is('span')) {
      Events.trigger('newSuggestedSearch', $(event.target).parents('.suggestion').text());
    } else {
      Events.trigger('newSuggestedSearch', $(event.target).text());
    }
  },
  render: function() {
    var suggestions = this.props.suggestions.map((suggestion) =>
      <div className="suggestion" onClick={this.handleClickSuggestion} data-state={suggestion.state}>
        <input type="checkbox"/>{suggestion.queryText}
      </div>);

    if (suggestions.length === 0) {
      return null;
    } else {
      return <div className="suggestions">
        <h3>Popular queries for this package</h3>
        <div className="suggestionList">
          {suggestions}
        </div>
      </div>;
    }
  },
});

module.exports = ReferenceTool;
