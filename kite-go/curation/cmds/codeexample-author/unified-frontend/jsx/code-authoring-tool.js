/* ## Code Authoring Tool ## */

React = require('react/addons');

require('./global-utils.js');

CodeExampleEditor = require('./code-example-editor.js');
CheckboxFilter = require('./checkbox-filter.js');
EditorModes = require('./example-editor-modes.js');

if (!Array.prototype.findIndex) {
  Array.prototype.findIndex = function(predicate) {
    if (this === null) {
      throw new TypeError('Array.prototype.findIndex called on null or undefined');
    }
    if (typeof predicate !== 'function') {
      throw new TypeError('predicate must be a function');
    }
    var list = Object(this);
    var length = list.length >>> 0;
    var thisArg = arguments[1];
    var value;

    for (var i = 0; i < length; i++) {
      value = list[i];
      if (predicate.call(thisArg, value, i, list)) {
        return i;
      }
    }
    return -1;
  };
}

function Set() {
  this.members = [];

  this.add = function(obj) {
    if (!this.has(obj)) {
      this.members.push(obj);
    }
  };

  this.has = function(obj) {
    return this.members.indexOf(obj) > -1;
  };

  this.remove = function(obj) {
    var idx = this.members.indexOf(obj);
    if (idx > -1) {
      this.members.splice(idx, 1);
    }
  };

  this.count = function() {
    return this.members.length;
  };
}

var UnsavedSnippets = new Set();

window.onbeforeunload = function(e) {
  if (UnsavedSnippets.count() > 0) {
    return 'There are unsaved code examples, which will be lost if you leave this page.';
  }
};

function makeIdGenerator() {
  var nextId = 0;
  return () => nextId++;
}

var exampleKeyGen = makeIdGenerator();

// from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/includes
if (![].includes) {
  Array.prototype.includes = function(searchElement /*, fromIndex*/ ) {
    'use strict';
    var O = Object(this);
    var len = parseInt(O.length) || 0;
    if (len === 0) {
      return false;
    }
    var n = parseInt(arguments[1]) || 0;
    var k;
    if (n >= 0) {
      k = n;
    } else {
      k = len + n;
      if (k < 0) {k = 0;}
    }
    var currentElement;
    while (k < len) {
      currentElement = O[k];
      if (searchElement === currentElement ||
         (searchElement !== searchElement && currentElement !== currentElement)) {
        return true;
      }
      k++;
    }
    return false;
  };
}

var ExampleStates = ['in_progress', 'pending_review', 'needs_attention', 'approved'];

var CodeAuthoringTool = React.createClass({
  getInitialState: function() {
    return {
      loading: true,
      examples: [],
      filter: ExampleStates.slice(),
      lockOwner: '',
      lockExpiration: this.props.initLockExpiration || -1,
    };
  },
  loadAndLock: function() {
    $.ajax({
      type: 'GET',
      url: ['/api', this.props.lang, this.props.pkg, 'lockAndList'].join('/'),
      dataType: 'json',
      success: (data) => {
        var examplesWithKeys = data.examples.map((example) => {
          example.key = exampleKeyGen();
          return example;
        });
        this.setState({
          loading: false,
          pkg: this.props.pkg,
          examples: examplesWithKeys,
          lockOwner: data.accessLock.userEmail,
          lockExpiration: data.accessLock.expiration,
        });
        console.log(data);
      },
      error: (data) => {
        console.log(arguments);
      },
    });
  },
  componentDidMount: function() {
    this.loadAndLock();
  },
  newExample: function() {
    var newExample = emptyExample(this.props.lang, this.props.pkg);
    newExample.key = exampleKeyGen();
    this.setState({ examples: this.state.examples.concat([newExample]) });
  },
  clone: function(position, clone) {
    clone.key = exampleKeyGen();
    clone.backendId = -1; // TODO make this null
    var examplesWithClone = this.state.examples.slice();
    examplesWithClone.splice(position+1, 0, clone);
    this.setState({ examples: examplesWithClone });
  },
  remove: function(position) {
    UnsavedSnippets.remove(this.state.examples[position].key);
    var examplesAfterRemoval = this.state.examples.slice();
    examplesAfterRemoval.splice(position, 1);
    this.setState({ examples: examplesAfterRemoval });
  },
  updateStatus: function(position, newStatus) {
    var updatedExamples = this.state.examples.slice();
    updatedExamples[position].status = newStatus;
    this.setState({ examples: updatedExamples });
  },
  changeFilter: function(newFilter) {
    this.setState({ filter: newFilter });
  },
  setReadOnly: function() {
    // something of a hack: lockOwner is supposed to be the email address of the
    // current user, when we call setReadOnly we might not know that, so we set it
    // to an invalid email address instead:
    this.setState({ lockOwner: 'someone else' });
  },
  render: function() {
    console.log('re-rendering', this.state.examples);
    if (this.state.loading) {
      return <div>
        <h1>Code Examples</h1>
        <p>loading</p>
      </div>;
    } else {
      var readOnly = this.props.userEmail !== this.state.lockOwner;
      return <div className="authoring-wrapper">
        <h1>
          <div className="edit-state">
            {readOnly ?
              '(read-only while ' + this.state.lockOwner + ' edits this)' :
              this.state.lockOwner }
          </div>
          Code Examples
        </h1>
        <div className="scroll-container">
          <Breadcrumbs lang={this.props.lang} pkg={this.props.pkg} />
          <CheckboxFilter
            onFilterChanged={this.changeFilter}
            options={ExampleStates} />
          {this.state.examples.map((example, position) =>
            this.state.filter.includes(example.status) &&
              <CodeExampleEditor
                key={example.key}
                lang={this.props.lang}
                pkg={this.props.pkg}
                initExample={example}
                position={position}
                clone={this.clone}
                remove={this.remove}
                updateStatus={this.updateStatus}
                unsavedSnippets={UnsavedSnippets}
                setReadOnly={this.setReadOnly}
                mode={readOnly ? EditorModes.READONLY : EditorModes.EDIT} />)}
          {!readOnly &&
            <button onClick={this.newExample} className='new-code-example'>
              <div className='icon'>+</div>New code example
            </button>}
          <div className="footer">
            <p><a href="https://quip.com/XhbBAdtFv810">Package wikis</a></p>
            <p><span className="glyph-icon">⚛</span></p>
          </div>
        </div>
      </div>;
    }
  },
});

function emptyExample(language, pkg) {
  return {
    backendId: -1, // TODO make this null
    language: language,
    package: pkg,
    title: '',
    status: 'in_progress',
    prelude: '',
    code: '',
    postlude: '',
    output: '',
    tags: [],
    comments: [],
  };
}

var Breadcrumbs = React.createClass({
  render: function() {
    return <div className='breadcrumbs'>
      <span>{this.props.lang}</span><span>{this.props.pkg}</span>
    </div>;
  },
});

module.exports = CodeAuthoringTool;
