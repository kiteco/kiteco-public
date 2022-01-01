// This JS file contains logic strictly to the container, and not to
// the Reference or Code Authoring tools. The following logic only has
// to do with resizing the bar that separates the two sections, some URL
// parameters, and fullscreen.

$ = require('jquery');
require('jquery-ui/resizable');

var ContainerLogic = function() {

  var urlParams;

  var requestFullScreen = function(element) {
    // Supports most browsers and their versions.
    var requestMethod = element.requestFullScreen || element.webkitRequestFullScreen || element.mozRequestFullScreen || element.msRequestFullscreen;

    if (requestMethod) { // Native full screen.
      requestMethod.call(element);
    } else if (typeof window.ActiveXObject !== "undefined") { // Older IE.
      var wscript = new ActiveXObject("WScript.Shell");
      if (wscript !== null) {
        wscript.SendKeys("{F11}");
      }
    }
  };


  var _initializeContainerLogic = function() {


    $('body').on('keydown', function(event) {
      if (event.keyCode===70 && event.altKey) {
        requestFullScreen(document.body);
      } else if (event.keyCode===83 && event.altKey) {
        event.preventDefault();
        if (StackoverflowViewer !== undefined ) {
          Events.trigger('openStackoverflowviewer');
        }
      }
    });

    (window.onpopstate = function () {
      var match,
          pl     = /\+/g,  // Regex for replacing addition symbol with a space
          search = /([^&=]+)=?([^&]*)/g,
          decode = function (s) { return decodeURIComponent(s.replace(pl, " ")); },
          query  = window.location.search.substring(1);

      urlParams = {};
      while ((match = search.exec(query)))
        urlParams[decode(match[1])] = decode(match[2]);
    })();

    if (urlParams.onlyReference === "true" || urlParams.onlyReference === 'true/') {
      $('#referenceTool').width('60%');
      $('#codeAuthoringTool').css({
        'display': 'none',
        'visibility': 'hidden',
      });
    } else {
      $('#referenceTool').resizable({
        handles: 'e',
        resize: function(event, ui) {
          $('#codeAuthoringTool').width($('body').width()-ui.size.width);
        },
      });
    }
  };


  return {
    initializeContainerLogic : _initializeContainerLogic,
    urlParams : urlParams,
  };
}();


$(document).ready(function() {
  ContainerLogic.initializeContainerLogic();
});
