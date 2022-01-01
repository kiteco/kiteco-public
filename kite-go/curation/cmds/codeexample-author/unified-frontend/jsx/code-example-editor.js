React = require('react/addons');

require('./global-utils.js');

EditorModes = require('./example-editor-modes.js');
AnnotatedExample = require('../../../../../../js-lib/AnnotatedExample.js');

function sliceAttributes(obj /*, attr1, attr2, attr3, ...*/) {
  var r = {};
  Array.prototype.slice.call(arguments, 1).forEach(function(name) {
    r[name] = obj[name];
  });
  return r;
}

function extend(a, b) {
  for (var p in b) {
    if (b.hasOwnProperty(p)) {
      a[p] = b[p];
    }
  }
  return a;
}

function isElementInViewport(elt) {
  var rect = elt.getBoundingClientRect();
  var windowBottom = (window.innerHeight || document.documentElement.clientHeight);
  return (rect.top >= 0 && rect.top <= windowBottom) ||
         (rect.bottom >= 0 && rect.bottom <= windowBottom);
}

var CodeExampleEditor = React.createClass({
  saveDelay: 1000,
  scrollTimeout: -1,
  getInitialState: function() {
    return extend(
      sliceAttributes(this.props.initExample, 'title', 'prelude', 'code', 'postlude', 'backendId', 'comments', 'status'),
      {
        empty: this.props.initExample.title === '' &&
               this.props.initExample.prelude === '' &&
               this.props.initExample.code === '' &&
               this.props.initExample.postlude === '',
        saveTimeout: -1,
        deleting: false,
        executionSuccess: true,
        lintWarnings: [],
        annotatedSegments: [],
      });
  },
  componentDidMount: function() {
    var domNode = React.findDOMNode(this);
    (this.props.scrollElt || domNode.parentElement).addEventListener('scroll', (event) => {
      window.clearTimeout(this.scrollTimeout);
      this.scrollTimeout = window.setTimeout(() => {
        this.setState({ onscreen: isElementInViewport(domNode) });
      }, 0);
    });
    this.setState({ onscreen: isElementInViewport(domNode) });
  },
  isReadOnly: function() {
    return this.props.mode === EditorModes.READONLY;
  },
  isEdit: function() {
    return this.props.mode === EditorModes.EDIT;
  },
  isEditOrModerate: function() {
    return this.props.mode === EditorModes.MODERATE || this.props.mode === EditorModes.EDIT;
  },
  scheduleSave: function(newState) {
    newState = newState || {};
    window.clearTimeout(this.state.saveTimeout);
    this.setState(extend({
      empty: false,
      saveTimeout: window.setTimeout(this.save, this.saveDelay),
    }, newState));
  },
  changeTitle: function(event) {
    if (this.isEdit()) {
      this.scheduleSave({ title: event.target.value });
    }
  },
  changePrelude: function(newPrelude) {
    this.scheduleSave({ prelude: newPrelude });
  },
  changeCode: function(newCode) {
    this.scheduleSave({ code: newCode });
  },
  changePostlude: function(newPostlude) {
    this.scheduleSave({ postlude: newPostlude });
  },
  changeComments: function(newComments) {
    this.scheduleSave({ comments: newComments });
  },
  changeStatus: function(event) {
    if (this.isEditOrModerate()) {
      console.log("scheduling save with new status:", event.target.value);
      this.scheduleSave({ status: event.target.value });
      map(this.props.updateStatus, f => f(this.props.position, event.target.value));
    }
  },
  getData: function() {
    return {
      language: this.props.initExample.language,
      package: this.props.initExample.package,
      backendId: this.state.backendId,
      frontendId: 0, // TODO remove this from backend
      prelude: this.state.prelude,
      code: this.state.code,
      postlude: this.state.postlude,
      output: this.state.output,
      title: this.state.title,
      comments: this.refs.commentThread.getComments(),
      status: this.state.status,
    };
  },
  save: function() {
    if (!this.props.pkg) {
      console.error("example missing package, cannot save:", this.props, this.state);
      return;
    }
    if (this.isEditOrModerate()) {
      console.log('saving', this.getData());
      $.ajax({
        type: this.state.backendId === -1 ? 'POST' : 'PUT',
        url: this.state.backendId === -1 ?
          '/' + ['api', this.props.lang, this.props.pkg, 'examples'].join('/') :
          '/' + ['api', 'example', this.state.backendId].join('/'),
        dataType: 'json',
        data: JSON.stringify(this.getData()),
        success: (data) => {
          console.log('saved:', data);
          if (this.props.unsavedSnippets && this.state.backendId === -1) {
            this.props.unsavedSnippets.remove(-1);
          }
          this.setState({
            backendId: data.backendId,
            comments: data.comments,
            saveTimeout: -1,
          });
        },
        error: (data) => {
          if (data.responseJSON && data.responseJSON.code) {
            switch (data.responseJSON.code) {
            case 8:
              alert("sorry, somebody else is editing this package (" + this.props.initExample.package + ") now; try again later");
              maybeCall(this.props.setReadOnly);
              return;
            }
          }
          // default case (either no app error code, or unmatched app error code)
          console.error('save failed, retrying in ' + this.saveDelay/1000 + ' seconds...');
          this.scheduleSave();
        },
      });
    }
  },
  clone: function() {
    map(this.props.clone, f => f(this.props.position, this.getData()));
  },
  comment: function() {
    this.refs.commentThread.addComment();
  },
  del: function() {
    if (this.state.backendId === -1) {
      map(this.props.remove, f => f(this.props.position));
    } else if (this.isEdit()) {
      this.setState({ deleting: true });
      $.ajax({
        type: 'PUT',
        url: '/' + ['api', 'example', this.state.backendId].join('/'),
        dataType: 'json',
        data: JSON.stringify(extend(this.getData(), {status: 'deleted'})),
        success: (data) => {
          console.log('deleted:', data);
          this.props.remove(this.props.position);
        },
        error: (data) => {
          // TODO: better error handling here
          console.error('failed to delete', data.responseJSON, this.props, this.state);
          this.setState({ deleting: false });
        },
      });
    }
  },
  run: function() {
    $.ajax({
      type: 'POST',
      url: '/' + ['api', this.props.lang, 'execute'].join('/'),
      data: {
        title: this.state.title,
        backendId: this.state.backendId,
        prelude: this.state.prelude,
        code: this.state.code,
        postlude: this.state.postlude,
      },
      success: (body) => {
        var data = JSON.parse(body);
        console.log('exe:', data);
        this.setState({
          executionSuccess: data.succeeded,
          lintWarnings: data.style_violations.concat(data.title_violations),
          annotatedSegments: data.prelude_segments.concat(data.code_segments).concat(data.postlude_segments),
        });
        /* commented out for now since at the moment the backend doesn't actually store execution output
        if (data.succeeded) {
          this.scheduleSave();
        }
        */
      },
      error: (data) => {
        console.error('failed execution:', data);
      },
    });
  },
  focusPrelude: function() {
    this.refs.preludeEditor.focus();
  },
  focusCode: function() {
    this.refs.codeEditor.focus();
  },
  focusPostlude: function() {
    this.refs.postludeEditor.focus();
  },
  render: function() {
    var ex = this.props.initExample;

    // not a huge fan of this:
    var saveIndicator;
    if (this.state.empty) {
      saveIndicator = '';
    } else if (!this.state.executionSuccess) {
      map(this.props.unsavedSnippets, x => x.add(this.state.backendId));
      saveIndicator = '(not saving until error is resolved)';
    } else if (this.state.saveTimeout === -1) {
      map(this.props.unsavedSnippets, x => x.remove(this.state.backendId));
      saveIndicator = 'saved';
    } else {
      map(this.props.unsavedSnippets, x => x.add(this.state.backendId));
      saveIndicator = 'editing';
    }

    var editorClass = 'example-editor';
    if (this.state.deleting) {
      editorClass += ' deleting';
    }
    if (!this.state.executionSuccess) {
      editorClass += ' execution-error';
    }

    // Note: the placeholder=' ' below is a workaround for https://code.google.com/p/chromium/issues/detail?id=401185
    return <div>
      <div className='workflow-pane'>
        <select value={this.state.status} onChange={this.changeStatus}>
          <option value='in_progress'>In progress</option>
          <option value='pending_review'>Pending review</option>
          <option value='needs_attention'>Needs attention</option>
          <option value='approved'>Approved</option>
        </select>
      </div>
      <div className='main-pane'>
        <div className={editorClass}>
          <label className='title'><span>TITLE</span> <input placeholder=' ' type='text' value={this.state.title} onChange={this.changeTitle} /></label>
          {this.state.lintWarnings.length > 0 &&
            <div className="lint-warnings">
              {map(this.state.lintWarnings, warning =>
                <div>{warning.message} (line {warning.line})</div>
              )}
            </div>}
          <AceEditor
            ref='preludeEditor'
            lang={this.props.lang}
            onChange={this.changePrelude}
            extraClass='prelude'
            initContent={ex.prelude}
            onscreen={this.state.onscreen}
            runCode={this.run}
            nextEditor={this.focusCode}
            readOnly={!this.isEdit()} />
          <AceEditor
            ref='codeEditor'
            lang={this.props.lang}
            onChange={this.changeCode}
            extraClass='code'
            initContent={ex.code}
            onscreen={this.state.onscreen}
            runCode={this.run}
            prevEditor={this.focusPrelude}
            nextEditor={this.focusPostlude}
            readOnly={!this.isEdit()} />
          <AceEditor
            ref='postludeEditor'
            lang={this.props.lang}
            onChange={this.changePostlude}
            extraClass='postlude'
            initContent={ex.postlude}
            onscreen={this.state.onscreen}
            runCode={this.run}
            prevEditor={this.focusCode}
            readOnly={!this.isEdit()} />
          {this.state.annotatedSegments.length > 0 &&
            <AnnotatedExample segments={this.state.annotatedSegments} style={{margin: 0}} />}
          <CommentThread ref='commentThread' initialComments={this.state.comments} onChange={this.changeComments} />
        </div>
        <div className='editor-toolbar'>
          <span className='save-indicator'>{saveIndicator}</span>
          {this.isEdit() && <span>
              <button onClick={this.run} className='run'>run</button>
              <button onClick={this.clone} className='clone'>clone</button>
              <button onClick={this.comment} className='comment'>comment</button>
              <button onClick={this.del} className='delete'>delete</button>
            </span>}
        </div>
      </div>
    </div>;
  },
});

// Backend uses seconds; Date.now() uses milliseconds
function nowSeconds() {
  return Math.floor(Date.now()/1000);
}

var CommentThread = React.createClass({
  getInitialState: function() {
    return { comments: this.props.initialComments };
  },
  componentWillReceiveProps: function(nextProps) {
    this.setState({ comments: nextProps.initialComments });
  },
  // called by CodeExampleEditor:
  addComment: function(evt) {
    this.setState({ comments: this.state.comments.concat([{
      backendId: -1,
      text: this.state.newComment,
      createdBy: INIT_DATA.userEmail || '',
      createdAt: nowSeconds(),
      dismissed: 0,
    }])});
  },
  // called by CodeExampleEditor:
  getComments: function(evt) {
    return this.state.comments;
  },
  changeComment: function(changedComment) {
    var commentIndex = this.state.comments.findIndex((comment) =>
      comment.backendId === changedComment.backendId);
    var newComments = this.state.comments.slice();
    newComments.splice(commentIndex, 1, changedComment);
    this.setState({ comments: newComments });
    this.props.onChange(newComments);
  },
  render: function() {
    return <div>
      {map(this.state.comments, (comment) =>
        <SingleComment
          initBackendId={comment.backendId}
          initText={comment.text}
          initDismissed={comment.dismissed}
          createdBy={comment.createdBy}
          createdAt={comment.createdAt}
          onChange={this.changeComment} />
      )}
    </div>;
  },
});

function pluralize(s, n) {
  if (Math.abs(n) === 1) {
    return s;
  } else {
    return s + 's';
  }
}

function humaneDateDiff(a, b) {
  if (typeof a.getTime === 'function') {
    a = a.getTime();
  }
  if (typeof b.getTime === 'function') {
    b = b.getTime();
  }
  if (a === 0 || b === 0) {
    // If either date is zero, it's more likely to be a bug due to Go's zero
    // values rather than a real date, so display nothing:
    return '';
  }
  var diff = a - b;

  units = [
    [60 * 60 * 24 * 365, 'year'],
    [60 * 60 * 24 * 7 * 4, 'month'],
    [60 * 60 * 24 * 7, 'week'],
    [60 * 60 * 24, 'day'],
    [60 * 60, 'hour'],
    [60, 'minute'],
  ];

  for (var i = 0; i < units.length; i++) {
    var unit = units[i][0];
    var label = units[i][1];
    unit_diff = Math.round(diff / unit);
    if (unit_diff !== 0) {
      var suffix = diff > 0 ? 'from now' : 'ago';
      return [Math.abs(unit_diff), pluralize(label, unit_diff), suffix].join(' ');
    }
  }

  return 'just now';
}

var SingleComment = React.createClass({
  getInitialState: function() {
    return {
      backendId: this.props.initBackendId,
      text: this.props.initText,
      dismissed: this.props.initDismissed,
    };
  },
  componentWillReceiveProps: function(nextProps) {
    this.setState({
      backendId: nextProps.initBackendId,
    });
  },
  change: function(evt) {
    this.setState({ text: evt.target.value });
    this.props.onChange({
      backendId: this.state.backendId,
      text: evt.target.value,
      createdBy: this.props.createdBy,
      createdAt: this.props.createdAt,
      dismissed: this.state.dismissed,
    });
  },
  componentDidMount: function() {
    if (this.props.initBackendId === -1) {
      React.findDOMNode(this.refs.textarea).focus();
    }
    this.delayedResize();
  },
  componentDidUpdate: function() {
    this.delayedResize();
  },
  delayedResize: function() {
    window.setTimeout(this.resize, 0);
  },
  resize: function() {
    elt = React.findDOMNode(this.refs.textarea);
    elt.style.height = 'auto';
    elt.style.height = elt.scrollHeight+'px';
  },
  clickDismiss: function() {
    if (this.state.dismissed === 0) {
      this.setState({ dismissed: 1 });
    } else {
      this.setState({ dismissed: 0 });
    }
    this.props.onChange({
      backendId: this.state.backendId,
      text: this.state.text,
      createdBy: this.props.createdBy,
      createdAt: this.props.createdAt,
      dismissed: this.state.dismissed,
    });
  },
  render: function() {
    var dismissClass = (this.state.dismissed === 0 ? 'unresolved' : 'dismissed');
    return <div className={'comment ' + dismissClass}>
      <div className="metadata">
        <button className={'dismiss ' + dismissClass} onClick={this.clickDismiss}>
          {this.state.dismissed === 0 ? 'dismiss' : '✓ dismissed'}
        </button>
        <cite>{this.props.createdBy} </cite>
        <time>{humaneDateDiff(this.props.createdAt, nowSeconds())}</time>
        {this.props.backendId}
      </div>
      <textarea
        ref="textarea"
        rows="1"
        value={this.state.text}
        onChange={this.change} />
    </div>;
  },
});

var AceEditor = React.createClass({
  initAceEditor: function() {
    this.editor = ace.edit(React.findDOMNode(this));
    this.editor.getSession().setMode('ace/mode/'+this.props.lang);
    this.editor.setTheme('ace/theme/cobalt');
    this.editor.setOptions({maxLines: 40});
    this.editor.renderer.setShowGutter(false);
    this.editor.setShowPrintMargin(false);
    this.editor.setReadOnly(this.props.readOnly);
    this.editor.on('change', this.onChange);
    this.editor.commands.addCommand({
      name: 'run',
      bindKey: {win: "Ctrl-Enter|Shift-Enter", mac: "Ctrl-Enter|Shift-Enter"},
      exec: (editor) => {
        this.props.runCode();
      },
      readOnly: false,
    });
    this.editor.commands.addCommand({
      name: 'up1',
      bindKey: {win: "Up|Ctrl-P", mac: "Up|Ctrl-P"},
      exec: (editor, args) => {
        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateUp(args.times);
        var lineAfter = editor.getSelectionRange().start.row;
        if (lineBefore === lineAfter && typeof this.props.prevEditor === 'function') {
          this.props.prevEditor();
        }
      },
      readOnly: true,
    });
    this.editor.commands.addCommand({
      name: 'down1',
      bindKey: {win: "Down|Ctrl-N", mac: "Down|Ctrl-N"},
      exec: (editor, args) => {
        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateDown(args.times);
        var lineAfter = editor.getSelectionRange().start.row;
        if (lineBefore === lineAfter && typeof this.props.nextEditor === 'function') {
          this.props.nextEditor();
        }
      },
      readOnly: true,
    });
  },
  componentDidMount: function() {
    if (!this.editor && this.props.onscreen) {
      this.initAceEditor();
    }
  },
  componentDidUpdate: function() {
    if (!this.editor && this.props.onscreen) {
      this.initAceEditor();
    }
    map(this.editor, e => e.setReadOnly(this.props.readOnly));
  },
  onChange: function() {
    if (typeof this.props.onChange === 'function') {
      this.props.onChange(this.editor.getValue());
    }
  },
  focus: function() {
    if (!this.editor) {
      this.initAceEditor();
    }
    this.editor.focus();
  },
  render: function() {
    return <div className={'ace-wrapper ' + this.props.extraClass}>{this.props.initContent}</div>;
  },
});

module.exports = CodeExampleEditor;
