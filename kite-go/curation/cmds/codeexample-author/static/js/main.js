var CodeEditors = function() {

  var DEFAULT_NUM_EXAMPLES = 5;

  var sessionArgs = null;

  var codeExampleCounter = 0;

  // These are all Ace Editor objects created
  var allAceEditors = [];

  var _initializeEditors = function(args) {
    sessionArgs = typeof args === 'undefined' ? '' : args;

    /*  Let's detect first if we're fetching existing code examples
      or creating empty ones from scratch
    */
    if ($('body').attr('data-fetch-examples') === 'true') {
      fetchExamplesFromServer($('body').attr('data-package'), $('body').attr('data-language'));
    } else {
      createEmptyExamples(DEFAULT_NUM_EXAMPLES);
    }
  };

  var createEmptyExamples = function(n) {
    for (var i = 0; i < n; i++) {
      _createCodeExample();
    }
  };

  var _addComment = function($codeExample, text, backendId) {
    if (typeof text === 'undefined') {
      text = '';
    }
    if (typeof backendId === 'undefined') {
      backendId = -1;
    }
    $newComment = $('<div class="comment" data-backend-id="' + backendId + '" placeholder="Comment" contenteditable>'+text+'</div>');
    $codeExample.find('.comments').append($newComment);
    $newComment.focus();
  };

  var _createCodeExample = function(data, $siblingBefore) {
    var $codeExample = $(
      '<div class="codeExample" data-deleted="false" data-editor-id="' + codeExampleCounter +'">' +
      '  <div class="format">Autoformat</div>' +
      '  <div class="add cloneButton">Clone</div>' +
      '  <div class="add commentButton">Comment</div>' +
      '  <div class="delete first">' +
      '    <div class="firstStep"> - </div>' +
      '    <div class="secondStep"><div class="confirm">Delete</div><div class="cancel">Cancel</div></div>' +
      '  </div>' +
      '  <ul class="tags" placeholder="Tags"></ul>' +
      '  <div class="title" contenteditable></div>' +
      '  <div class="title_warnings hidden"><span></span><div class="close_warnings">&#10005;</div></div>' +
      '  <div class="prelude" id="prelude-' + codeExampleCounter +'"></div>' +
      '  <div class="code" id="code-' + codeExampleCounter +'"></div>' +
      '  <div class="postlude" id="postlude-' + codeExampleCounter + '"></div>' +
      '  <div class="output"><div class="play">Run &#9654;</div><div class="outputContent"></div></div>' +
      '  <div class="comments"></div>' +
      '</div>');
    codeExampleCounter++;

    if (typeof data !== 'undefined') {
      if(typeof data.tags !== 'undefined' && data.tags !== null) {
        for (var i = 0; i < data.tags.length; i++) {
          $codeExample.find('.tags').append('<li>' + data.tags[i] +'</li>');
        }
      }
      $codeExample.find('.title').text(data.title);
      $codeExample.find('.prelude').text(data.prelude);
      $codeExample.find('.code').text(data.code);
      $codeExample.find('.postlude').text(data.postlude);

      if (typeof data.colorized_output === 'undefined') {
        $codeExample.find('.outputContent').text(data.output);
      } else {
        $codeExample.find('.outputContent').html(data.colorized_output);
      }

      if(typeof $siblingBefore === 'undefined') {
        $codeExample.attr('data-backend-id', data.backendId);
      }

      if (typeof data.comments !== 'undefined') {
        for (var i = 0; i < data.comments.length; i++) {
          comment = data.comments[i];
          _addComment($codeExample, comment.text, comment.backendId);
        }
      }
    } else {
      console.log('A blank new code example has been created');
      $codeExample.attr('data-new-code-example', true);
    }

    if(typeof $siblingBefore !== 'undefined') {
      $codeExample.attr('data-new-code-example', true);
      $siblingBefore.after($codeExample);
    } else {
      $('.codeExampleContainer').append($codeExample);
    }
    createAceEditorForCodeExample($codeExample);
  };

  var createAceEditorForCodeExample = function($codeExample) {
    args = sessionArgs;

    var editorId = parseInt($codeExample.attr('data-editor-id'));

    var codeExampleEditor = {
      editorId : editorId,
      preludeEditor : ace.edit('prelude-' + editorId),
      codeEditor: ace.edit('code-' + editorId),
      postludeEditor: ace.edit('postlude-' + editorId),
    };

    var theme = args.theme ? args.theme : 'ace/theme/cobalt';
    var mode = args.mode ? args.mode :  'ace/mode/python';
    var showPrintMargin = args.showPrintMargin ? args.showPrintMargin : false;
    var showLineNumbers = args.showLineNumbers ? args.showLineNumbers : false;

    var setSettingsForEditor = function(editor) {
      editor.setTheme(theme);
      editor.setShowPrintMargin(false);
      editor.renderer.setShowGutter(false);
      editor.getSession().setMode(mode);
      editor.setOptions({maxLines: 40});
    };

    setSettingsForEditor(codeExampleEditor.preludeEditor);
    setSettingsForEditor(codeExampleEditor.codeEditor);
    setSettingsForEditor(codeExampleEditor.postludeEditor);

    codeExampleEditor.preludeEditor.commands.addCommand({
      name: 'down1',
      bindKey: {
        mac: "Down",
        win: "Down|Ctrl-N",
      },
      exec: function(editor, args) {
        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateDown(args.times);
        var lineAfter = editor.getSelectionRange().start.row;

        if (lineBefore === lineAfter) {
          codeExampleEditor.codeEditor.focus();
          codeExampleEditor.postludeEditor.navigateFileStart();
        }
      },
      multiSelectAction: "forEach",
      scrollIntoView: "cursor",
      readOnly: true,
    });

    codeExampleEditor.codeEditor.commands.addCommand({
      name: 'down2',
      bindKey: {
        mac: "Down",
        win: "Down|Ctrl-N",
      },
      exec: function(editor, args) {

        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateDown(args.times);
        var lineAfter = editor.getSelectionRange().start.row;

        if (lineBefore === lineAfter) {
          codeExampleEditor.postludeEditor.focus();
          codeExampleEditor.postludeEditor.navigateFileStart();
        }
      },
      multiSelectAction: "forEach",
      scrollIntoView: "cursor",
      readOnly: true,
    });

    codeExampleEditor.codeEditor.commands.addCommand({
      name: 'up1',
      bindKey: {
        mac: "Up",
        win: "Up",
      },
      exec: function(editor, args) {

        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateUp(args.times);
        var lineAfter = editor.getSelectionRange().start.row;

        if (lineBefore === lineAfter) {
          codeExampleEditor.preludeEditor.focus();
          codeExampleEditor.preludeEditor.navigateFileEnd();
        }
      },
      multiSelectAction: "forEach",
      scrollIntoView: "cursor",
      readOnly: true,
    });

    codeExampleEditor.postludeEditor.commands.addCommand({
      name: 'up2',
      bindKey: {
        mac: "Up",
        win: "Up",
      },
      exec: function(editor, args) {

        var lineBefore = editor.getSelectionRange().start.row;
        editor.navigateUp(args.times);
        var lineAfter = editor.getSelectionRange().start.row;

        if (lineBefore === lineAfter) {
          codeExampleEditor.codeEditor.focus();
          codeExampleEditor.codeEditor.navigateFileEnd();
        }
      },
      multiSelectAction: "forEach",
      scrollIntoView: "cursor",
      readOnly: true,
    });

    allAceEditors.push(codeExampleEditor);
  };

  var createStateOfCodeExample = function(editorIndex) {
    var $editor = $('[data-editor-id="' + editorIndex + '"]');

    var backendId = $editor.attr('data-backend-id');
    if (typeof backendId === 'undefined') {
      backendId = -1;
    } else {
      backendId = parseInt(backendId);
    }

    var comments = [];
    $editor.find('.comment').each(function() {
      comments.push({
        backendId: parseInt($(this).attr('data-backend-id')),
        text: $(this).text(),
        dismissed: 0,
      });
    });

    var editorState = {
      backendId: backendId,
      frontendId: 0,
      prelude : allAceEditors[editorIndex].preludeEditor.getSession().getValue(),
      code : allAceEditors[editorIndex].codeEditor.getSession().getValue(),
      postlude : allAceEditors[editorIndex].postludeEditor.getSession().getValue(),
      output : $editor.find('.outputContent').text(),
      tags : [],
      title : $editor.find('.title').text(),
      comments: comments,
      status: 'in_progress',
      formatted_output: '',
    };

    return editorState;
  };

  var _createStateOfAllCodeExamples = function() {
    var codeExamples = [];

    for (var i = 0; i < allAceEditors.length; i++) {
      state = createStateOfCodeExample(i);
      state.order = codeExamples.length;

      // If the code example is empty then do not save it
      if ($.trim(state.code).length > 0 || state.id >= 0) {
        codeExamples.push(state);
      }
    }

    console.log("Saving code examples to server:");
    console.log(codeExamples);

    return {states: JSON.stringify(codeExamples)};
  };


  var _createStateOfNewCodeExamples = function() {
    var newCodeExamples = [];

    for (var i = 0; i < allAceEditors.length; i++) {
      state = createStateOfCodeExample(i);
      state.backendId = -1;
      // We get all code examples that: 1) Are new, and 2) Are modified.
      if ($('[data-editor-id="' + i + '"]').attr('data-new-code-example') === 'true' && $('[data-editor-id="' + i + '"]').attr('data-example-has-been-modified') === 'true' && $('[data-editor-id="' + i + '"]').attr('data-deleted') === 'false') {
        newCodeExamples.push(state);
      }
    }
    return newCodeExamples;
  };

  var _createStateOfModifiedCodeExamples = function() {
    var modifiedCodeExamples = [];

    for (var i = 0; i < allAceEditors.length; i++) {
      state = createStateOfCodeExample(i);
      // We get all code examples that: 1) Are modified, and 2) Are NOT new.
      if ($('[data-editor-id="' + i + '"]').attr('data-example-has-been-modified') === 'true' && $('[data-editor-id="' + i + '"]').attr('data-new-code-example') !== 'true') {
        state.status = ($('[data-editor-id="' + i + '"]').attr('data-deleted') === 'false') ? 'in_progress' : 'deleted';
        modifiedCodeExamples.push(state);
      }
    }
    return modifiedCodeExamples;
  };

  var _sendAllCodeExamples = function() {
    var allCodeExamples = _createStateOfAllCodeExamples();
    language = $('body').attr('data-language');
    pkg = $('body').attr('data-package');
    url = '/api/' + language + '/' + pkg + '/save';
    $.ajax({
      url: url,
      type: 'POST',
      dataType: 'json',
      data: allCodeExamples,
    })
    .done(function(data) {
      console.log("Received in response to save:");
      console.log(data);
      location.reload();
    })
    .fail(function() {
      console.log("Error in response to save");
    })
    .always(function() {
    });
  };

  var _save = function() {
    // Var inits
    var language = $('body').attr('data-language'),
        pkg = $('body').attr('data-package'),
        postUrl = '/api/' + language + '/' + pkg + '/examples',
        putUrl = '/api/example/',
        failure = false; // ...


    // First, we gather new code examples
    var newCodeExamples = _createStateOfNewCodeExamples();
    console.log(newCodeExamples);

    // Second, let's gather existing modified examples
    var existingModifiedExamples = _createStateOfModifiedCodeExamples();
    console.log(existingModifiedExamples);

    if (newCodeExamples.length === 0 && existingModifiedExamples.length === 0) {
      return;
    }

    // Now let's POST all new code examples, one by one.

    for (var i = newCodeExamples.length - 1; i >= 0; i--) { // Speaking of silly optimizations
      $.ajax({
        url: postUrl,
        type: 'POST',
        dataType: 'json',
        data: JSON.stringify(newCodeExamples[i]),
        async: false,
      })
      .done(function(resp) {
        console.log("POST: Successfully post new code example");
      })
      .fail(function() {
        console.log("POST: Error when trying to post a new code example");
        failure = true;
      });
    }

    // Now let's PUT modified code examples, one by one.

    for (var i = existingModifiedExamples.length - 1; i >= 0; i--) { // Speaking of silly optimizations
      $.ajax({
        url: putUrl + existingModifiedExamples[i].backendId,
        type: 'PUT',
        dataType: 'json',
        data: JSON.stringify(existingModifiedExamples[i]),
        async: false,
      })
      .done(function(resp) {
        console.log("PUT: Successfully updated existing code example");
      })
      .fail(function() {
        console.log("PUT: Error when trying to update an existing code example");
        failure = true;
      });
    }

    if (!failure) {
      location.reload();
    }



  };

  var fetchExamplesFromServer = function(pkg, language) {
    url = '/api/' + language + '/' + pkg + '/examples';
    console.log("fetching from: " + url);
    $.ajax({
      url: url,
      type: 'GET',
      dataType: 'json',
    })
    .done(function(response) {
      // Assumption: data returned by the server is an array of objects, each object has at least 4 fields corresponding
      // to prelude, code, postlude, and a stored output in the server
      console.log("Fetched data from backend:");
      console.log(response);
      for (var i = 0; i < response.length; i++) {
        _createCodeExample(response[i]);
      }
      // Add one empty example at the bottom
      _createCodeExample();
    })
    .fail(function() {
      console.log("Error fetching code examples");
    })
    .always(function() {
    });
  };

  var _autoformatCodeExample = function($codeExample) {
    var stateOfCodeExample = createStateOfCodeExample(parseInt($codeExample.attr('data-editor-id')));
    language = $('body').attr('data-language');
    url = '/api/' + language + '/autoformat';
    $.ajax({
      url: url,
      type: 'POST',
      dataType: 'json',
      data: stateOfCodeExample,
    })
    .done(function(response) {
      editorIndex = parseInt($codeExample.attr('data-editor-id'));
      allAceEditors[editorIndex].preludeEditor.getSession().setValue(response.prelude);
      allAceEditors[editorIndex].codeEditor.getSession().setValue(response.code);
      allAceEditors[editorIndex].postludeEditor.getSession().setValue(response.postlude);
      console.log("success formatting!!!!!");
    })
    .fail(function() {
      console.log("error");
    })
    .always(function() {
      console.log("complete");
    });
  };

  var _executeCodeExample = function($codeExample) {
    var stateOfCodeExample = createStateOfCodeExample(parseInt($codeExample.attr('data-editor-id')));
    language = $('body').attr('data-language');
    url = '/api/' + language + '/execute';
    $.ajax({
      url: url,
      type: 'POST',
      dataType: 'json',
      data: stateOfCodeExample,
    })
    .done(function(response) {
      console.log("done!");
      console.log(response.title_violations.length);
      console.log(response.output);

      if (typeof response.colorized_output === 'undefined' || response.colorized_output === '') {
        $codeExample.find('.outputContent').text(response.output);
      } else {
        $codeExample.find('.outputContent').html(response.colorized_output);
      }

      for (var i = 0; i < response.images.length; i++) {
        $img = $('<img alt="'+response.images[i].name+'"></img>');
        $img.attr('src', "data:image/png;base64," + response.images[i].data);
        $codeExample.find('.outputContent').append($img);
      }

      if (response.output === '' && response.succeeded) {
        if($codeExample.find('.executionMessage').length === 0) {
          $codeExample.find('.output').append('<div class="executionMessage"></div>');
        }
        $codeExample.find('.executionMessage').text("code example completed but generated no output");
      }

      // There are title violations.
      var title_warnings = "";

      for (var i = 0; i < response.title_violations.length; i++) {
        title_warnings = title_warnings + response.title_violations[i].message + "\n";
      }

      $codeExample.find('.title_warnings span').text(title_warnings);

      if (title_warnings.length > 0) {
        $codeExample.find('.title_warnings').removeClass('hidden');
      } else {
        $codeExample.find('.title_warnings').addClass('hidden');
      }

      if (!response.succeeded) {
        $codeExample.addClass('runFailed');
      } else {
        $codeExample.removeClass('runFailed');
      }

      var preludeAnnotations = [];
      var codeAnnotations = [];
      var postludeAnnotations = [];
      for (var i = 0; i < response.style_violations.length; i++) {
        var violation = response.style_violations[i];
        annotation = {
          row: violation.line,
          text: violation.message + ' (' + violation.rule_code + ')',
          type: "warning",
        };
        if (violation.segment === "prelude") {
          preludeAnnotations.push(annotation);
        } else if (violation.segment === "code") {
          codeAnnotations.push(annotation);
        } else if (violation.segment === "postlude") {
          postludeAnnotations.push(annotation);
        }
      }

      if (typeof response.errors !== 'undefined' && response.errors !== null) {
        for (var i = 0; i < response.errors.length; i++) {
          var error = response.errors[i];
          annotation = {
            row: error.line,
            text: error.message,
            type: "error",
          };
          if (error.segment === "prelude") {
            preludeAnnotations.push(annotation);
          } else if (error.segment === "code") {
            codeAnnotations.push(annotation);
          } else if (error.segment === "postlude") {
            postludeAnnotations.push(annotation);
          }
        }
      }


      var showGutter = preludeAnnotations.length + codeAnnotations.length + postludeAnnotations.length > 0;
      var editorIndex = parseInt($codeExample.attr('data-editor-id'));
      var editor = allAceEditors[editorIndex];

      // Show/hide the gutter according to whether there were any style violations
      editor.preludeEditor.renderer.setShowGutter(showGutter);
      editor.codeEditor.renderer.setShowGutter(showGutter);
      editor.postludeEditor.renderer.setShowGutter(showGutter);

      // Update editor contents with results of autoformat
      if (response.formatted_prelude !== null) {
        editor.preludeEditor.getSession().setValue(response.formatted_prelude);
      }
      if (response.formatted_code !== null) {
        editor.codeEditor.getSession().setValue(response.formatted_code);
      }
      if (response.formatted_postlude !== null) {
        editor.postludeEditor.getSession().setValue(response.formatted_postlude);
      }

      editor.preludeEditor.getSession().setAnnotations(preludeAnnotations);
      editor.codeEditor.getSession().setAnnotations(codeAnnotations);
      editor.postludeEditor.getSession().setAnnotations(postludeAnnotations);
    })
    .fail(function() {
      console.log("failed?");
      if($codeExample.find('.executionMessage').length === 0) {
        $codeExample.find('.output').append('<div class="executionMessage"></div>');
      }
      $codeExample.find('.executionMessage').text('The server is currently not running');
      console.log("error");
    })
    .always(function() {
      console.log("urg...complete");
    });
  };

  return {
    initializeEditors : _initializeEditors,
    createCodeExample : _createCodeExample,
    addComment : _addComment,
    createStateOfAllCodeExamples : _createStateOfAllCodeExamples,
    sendAllCodeExamples : _sendAllCodeExamples,
    executeCodeExample : _executeCodeExample,
    autoformatCodeExample : _autoformatCodeExample,
    createStateOfCodeExample : createStateOfCodeExample,
    save : _save,
  };
}();


$(document).ready(function() {
  CodeEditors.initializeEditors();

  $('.codeExampleContainer').on('click', '.codeExample', function(event) {
    if($(this).is(':last-child')) {
      CodeEditors.createCodeExample();
    }
  });

  $('#send').on('click', function(event) {
    event.preventDefault();
    CodeEditors.save();
  });

  $('.codeExampleContainer').on('click', '.codeExample', function(event) {
    $(this).attr('data-example-has-been-modified', true);
  });

  $('.codeExampleContainer').on('click', '.play', function(event) {
    event.preventDefault();
    CodeEditors.executeCodeExample($(this).parents('.codeExample'));
  });

  $('.codeExampleContainer').on('click', '.format', function(event) {
    event.preventDefault();
    CodeEditors.autoformatCodeExample($(this).parents('.codeExample'));
  });

  $('.codeExampleContainer').on('click', '.firstStep', function(event) {
    $(this).parent().removeClass('first').addClass('second');
  });

  $('.codeExampleContainer').on('click', '.secondStep .confirm', function(event) {
    console.log('DELETE EXAMPLE');
    $(this).parents('.codeExample').attr('data-deleted', 'true');
  });

  $('.codeExampleContainer').on('click', '.secondStep .cancel', function(event) {
    $(this).parents('.delete').removeClass('second').addClass('first');
  });

  $('.codeExampleContainer').on('keydown', '.title', function(event) {
    if (event.keyCode === 13) {
      event.preventDefault();
    }
  });
  $('.codeExampleContainer').on('click', '.cloneButton', function(event) {
    var codeState = CodeEditors.createStateOfCodeExample(parseInt($(this).parents('.codeExample').attr('data-editor-id')));
    CodeEditors.createCodeExample(codeState, $(this).parents('.codeExample'));
  });

  $('.codeExampleContainer').on('click', '.commentButton', function(event) {
    CodeEditors.addComment($(this).parents('.codeExample'));
  });

  $('.codeExampleContainer').on('click', '.close_warnings', function(event) {
    $(this).parents('.title_warnings').addClass('hidden');
  });
});
