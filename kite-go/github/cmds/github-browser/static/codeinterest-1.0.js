var CodeInterest = function() {

  // Global args and vars
  var GLOBAL_args, GLOBAL_isBookmarkingWindowOpen;

  // Global jQuery objects
  var $codeinterest_affordance, $bookmarkingWindow;


  var _initialize = function(args) {

    // Define global arguments for plugin
    if (!args || typeof args === 'undefined') {
      args = {
        selector: 'pre',
        innerCodeContainer : null,
      };
    } else {
      args.selector = (!args.selector || typeof args.selector === 'undefined') ? 'pre' : args.selector;
      args.innerCodeContainer = (!args.innerCodeContainer || typeof args.innerCodeContainer === 'undefined') ? null : args.innerCodeContainer;
    }

    // Set global args
    GLOBAL_args = args;

    // Set code interest affordance and append to body
    $codeinterest_affordance = $('<div id="codeinterest_affordance">&#9733;</div>').appendTo('body');

    $bookmarkingWindow = $('<div id="bookmarkingWindow"><div class="infoMessage">Code saved</div><div class="labelInstruction">Send to code authoring tool</div><div class="codeInput" contentEditable="true"></div><div class="commentInput" contenteditable data-placeholder="// Comments about this snippet"></div><div class="sendButton">Send</div></div>').appendTo('body');

    $codeinterest_affordance.on('mouseenter', function(event) {
      $(this).css({
        'opacity' : '1',
      });
    }).on('mouseout', function(event) {
      $(this).css({
        'opacity' : '0.6',
      });
    });

    $codeinterest_affordance.on('click', function(event) {
      event.preventDefault();
      showBookmarkingWindow($(this).parents(GLOBAL_args.selector), $(this));
    });

    $bookmarkingWindow.find('.sendButton').on('click', function(event) {

      sendStuffToCodeAuthoringUI();

    });

    // Attach listeners to body
    $('body').on('mouseenter', GLOBAL_args.selector, function(event) {
      event.stopPropagation();
      if(!GLOBAL_isBookmarkingWindowOpen) {
        $codeinterest_affordance.appendTo($(this)).css({
          'display' : 'block',
          'visibility' : 'visible',
        });
      }

    });

    $(document).mouseup(function (e) {
      if (GLOBAL_isBookmarkingWindowOpen) {
        var container = $bookmarkingWindow;
        if (!container.is(e.target) && container.has(e.target).length === 0) {
          hideBookmarkingWindow();
        }
      }
    });
  };


  var showBookmarkingWindow = function($snippetElement, $caller) {
    GLOBAL_isBookmarkingWindowOpen = true;
    $bookmarkingWindow.css({
      'display' : 'block',
      'visibility' : 'visible',
      'top' : $caller.offset().top,
      'left' : $caller.offset().left - $bookmarkingWindow.outerWidth() - 20,
    }).find('.codeInput').text((GLOBAL_args.innerCodeContainer!== null) ? $snippetElement.find(GLOBAL_args.innerCodeContainer).text() : $snippetElement.text());
  };

  var hideBookmarkingWindow = function() {
    GLOBAL_isBookmarkingWindowOpen = false;
    $bookmarkingWindow.css({
      'display' : 'none',
      'visibility' : 'hidden',
    });
  };

  var showSuccessSendingCode = function() {
    $bookmarkingWindow.addClass('info').find('.infoMessage').text('Snippet saved!');
    setTimeout(function() {
      hideBookmarkingWindow();
      $bookmarkingWindow.removeClass('info');
    }, 1200);
  };

  var showFailureSendingCode = function() {
    $bookmarkingWindow.addClass('info').find('.infoMessage').text('Server not responding!');
    setTimeout(function() {
      hideBookmarkingWindow();
      $bookmarkingWindow.removeClass('info');
    }, 1200);
  };

  var sendStuffToCodeAuthoringUI = function() {
    var pkg = (typeof urlParams.query === 'undefined') ? 'test' : urlParams.query.split('.')[0];
    var language = 'python';

    if (pkg === '') {
      throw "pkg not defined on the global scope";
    }

    $.ajax({
      url: 'http://curation.kite.com/api/' + language +'/' + pkg + '/externaladd',
      type: 'POST',
      data: {code: $bookmarkingWindow.find('.codeInput').text() + '\n#' + $bookmarkingWindow.find('.commentInput').text() },
    })
    .done(function(response) {
      console.log(response);
      showSuccessSendingCode();
    })
    .fail(function(response) {
      console.log(response);
      showFailureSendingCode();
    })
    .always(function() {
      console.log("complete");
    });

  };



  return {
    initialize : _initialize,
  };

}();
