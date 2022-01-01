/*

PARAM SEARCH

*/

var urlParams;
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

var ClusterNavigation = function() {
  var _fetchNavigation = function() {
    $.ajax({
      url: '/packages',
      type: 'GET',
      dataType: 'json',
    })
    .done(function(response) {
      var $pkg = $('<div class="packageMenu"></div>');
      $('#navigation .navigationContent').append($pkg);
      $pkg.append(drawMenu(response));
    })
    .fail(function() {
      console.log("error");
    })
    .always(function() {
      console.log("complete");
    });

    $('#navigation').on('click', '[data-type-nav="package"], [data-type-nav="submodule"]', function(event) {
      event.preventDefault();
      event.stopPropagation();
      if (typeof $(this).attr('data-open') === 'undefined') {
        fetchMoreLevels($(this));
        $(this).attr('data-open', 'true');
      } else {
        $(this).attr('data-open', $(this).attr('data-open') === 'true' ? 'false' : 'true');
      }
    });

    $('#navigation').on('click', '[data-type-nav="method"]', function(event) {
      event.preventDefault();
      event.stopPropagation();
      window.location.href = window.location.origin + '?query=' + $(this).find('.name').text() + '&numClusters=5';
    });
  };

  var fetchMoreLevels = function($requester) {
    $.ajax({
      url: $requester.attr('data-type-nav') === 'package' ? '/submodules' : '/methods' ,
      type: 'GET',
      data: $requester.attr('data-type-nav') === 'package' ? {packageName: $requester.find('.name').text()} : {submoduleName: $requester.find('.name').text()},
    })
    .done(function(response) {
      if (response.constructor !== Array) {
        response = JSON.parse(response);
      }
      $requester.append(drawMenu(response, $requester.attr('data-type-nav')));
    })
    .fail(function() {
      console.log("error");
    })
    .always(function() {
      console.log("complete");
    });
  };

  var drawMenu = function(packages, antecesor) {
    console.log(packages);
    var type = 'package';
    switch(antecesor) {
    case 'package':
      type = 'submodule';
      break;
    case 'submodule':
      type = 'method';
      break;
    default:
      break;
    }
    var $ul = $('<ul data-list-type="' + type +'"></ul>');
    var barShown = [false, false, false, false, false];
    var percBar = [0.25, 0.5, 0.75, 0.9, 1];
    for (var i = 0; i < packages.length; i++) {
      var freq = (parseFloat(packages[i].Freq) * 100).toFixed(2);
      if (packages[i].Freq <=  0.0001) {
        break;
      }
      $('<li data-type-nav=' + type +'></li>').append('<div class="label"><div class="name">' + packages[i].Name +'</div><div class="cdf">' + freq +'%</div></div>').appendTo($ul);
      for (var j = 4; j >= 0; j--) {
        if (packages[i].Cdf >= percBar[j] && !barShown[j]) {
          var cdf = (parseFloat(percBar[j]) * 100).toFixed(0);
          $('<div class="CdfSeparator"><div class="line"></div><span>' + cdf +'%<span></div>').appendTo($ul);
          for (var k = j; k >= 0; k--) {
            barShown[k] = true;
          }
          break;
        }
      }
    }
    return $ul;
  };

  return {
    fetchNavigation : _fetchNavigation,
  };

}();


var ClusterModel = function() {

}();

var ClusterListRepresentation = function() {

  var _buildAccordionWithData = function(appendToObject) {

    if(urlParams.query && urlParams.query.length>0) {
      $('.section.right').append('<div class="pageHeader"><div class="label">Exploring: </div><div class="packageName">' + urlParams.query +'</div></div>');
    }

    var dataToSend = {
      query: urlParams.query,
      numClusters: urlParams.numClusters,
    };

    console.log(dataToSend);
    $.ajax({
      url: '/clusters',
      type: 'POST',
      data: dataToSend,
    })
    .done(function(response) {
      var res = $.parseJSON(response);
      console.log(res);


      var $suggestedQueries = drawSuggestedQueries(appendToObject);
      var $accordionInvoc = drawClusters("Common invocation patterns", res.patternClusters, appendToObject);
      var $accordionCooccurent = drawClusters("Common co-occurent functions", res.cooccurClusters, appendToObject);
      var $accordionClusters = drawClusters("Clusters", res.codeClusters, appendToObject);

      if ($accordionCooccurent.length > 0) {
        highlightStatementFunctionsInAccordion($accordionCooccurent);
      }

      console.log("success");
    })
    .fail(function() {
      console.log("error");
    })
    .always(function() {
      console.log("complete");
    });
  };

  var drawSuggestedQueries = function(appendToObject) {
    console.log('draw suggested queries ------');
    $.ajax({
      url: 'http://curation.kite.com:8080/suggestions?query=' + urlParams.query,
      type: 'POST',
      async: false,
    })
    .done(function(response) {
      console.log('draw suggested queries ------ succ');
      $('<h2>'+'Suggested Queries'+'</h2>').appendTo(appendToObject);
      var $queryContainer = $('<div class="suggestedQueriesContainer"></div>');
      var $queryWrapper = $('<div id="queryWrapper"></div>');
      for (var i = 0; i < response.length; i++) {
        $('<a href="http://curation.kite.com:8090/stackoverflow?query=' + response[i] +'" class="suggestedQuery" target="_blank">' + response[i] +'</a>').appendTo($queryContainer);
      }
      $queryContainer.appendTo($queryWrapper);
      $queryWrapper.appendTo(appendToObject);
    })
    .fail(function() {
      console.log('draw suggested queries ------ err');
      console.log("error");
    })
    .always(function() {
      console.log('draw suggested queries ------ comp');
      console.log("complete");
    });
  };

  var highlightStatementFunctionsInAccordion = function($accordion) {
    var functionBeingExplored = urlParams.query.split('.')[1];
    $accordion.find('.sectionAccordion').each(function(index, el) {
      var re = new RegExp('\\s+', 'g');
      var statementFunctions = $(el).find('.title .statement').text().replace(re, '').split(',');
      $(el).find('.pln').each(function(index2, el2) {
        var txt = $(el2).text();
        for (var i = 0; i < statementFunctions.length; i++) {
          if (functionBeingExplored === txt) {
            $(el2).addClass('hl');
            break;
          } else if(statementFunctions[i] === txt) {
            $(el2).addClass('hg');
            break;
          }
        }
      });

    });
  };


  var drawClusters = function(title, data, appendToObject)  {
    if (data !== null && data.length > 0) {
      $('<h2>'+title+'</h2>' + (title === 'Clusters' ? '<div class="dropdownNumberClusters"><select><option value="3">3</option><option value="5">5</option><option value="10">10</option></select></div>' : '')).appendTo(appendToObject);
      if(!urlParams.query || urlParams.query.length===0) {
        $('.dropdownNumberClusters').remove();
      } else {
        if (!urlParams.numClusters || urlParams.numClusters.length===0) {
          urlParams.numClusters = 5;
        }
        $('.dropdownNumberClusters select').val(urlParams.numClusters);

        $('.dropdownNumberClusters').on('change', 'select', function(event) {
          event.preventDefault();
          window.location.href = window.location.origin + '?query=' + urlParams.query + '&numClusters=' + $(this).val();
        });

      }
      var $accordion = $('<div class="accordion"></div>');
      for (var i = 0; i < data.length; i++) {
        $accordion.append(drawSectionAccordion(data[i].representative.statement, '', data[i].snippets, data[i].percentage, i));
      }
      $accordion.appendTo(appendToObject);
      return $accordion;
    }
    return '';
  };

  var drawSectionAccordion = function(title, visibleStuff, snippets, percentage, clusterID) {

    var drawVisibleStuff = function(visibleStuff) {
      return $('<div class="visibleStuff">' + visibleStuff.replace(/(^\s+|\s+$)/g,'') +'</div>');
    };

    var drawSnippet = function(snippet, snippetID) {
      return $('<div class="snippet"><div class="snippetLabel">Snippet</div><div class="snippetContent snippetSt">' + snippet.statement.replace(/(^\s+|\s+$)/g,'') +'</div>' + '<div class="snippetContent snippetC" id =' + clusterID + '-' + snippetID + '>' + snippet.code.replace(/(^\s+|\s+$)/g,'') + '</div></div>');
    };

    var popAcc = parseFloat(percentage) * 100;
    var $sectionAccordion = $('<div class="sectionAccordion"></div>');
    var $title  = $('<div class="title">' + '<span class="popularity">' + popAcc.toFixed(2) + '%</span>' + '<span class="statement">' +  title + '</span>' +'</div>');
    var $content = $('<div class="content"></div>');
    $sectionAccordion.append($title);

    if (visibleStuff.length > 0) {
      $sectionAccordion.append(drawVisibleStuff(visibleStuff));
    }

    for (var i = 0; i < snippets.length; i++) {
      $content.append(drawSnippet(snippets[i], i));
    }

    $sectionAccordion.append($content);


    return $sectionAccordion;

  };





  return {
    buildAccordionWithData : _buildAccordionWithData,
  };

}();

var ClusterMapRepresentation = function() {

}();


var ClusterNotes = function() {

  var _addSnippetToNotesTray = function($snippet) {
    var $snippetCopy = $snippet.clone();
    $snippetCopy.append('<div class="snippetNotes" contenteditable></div>');
    $('#notesTray .listOfNotes').append($snippetCopy);

    $('#notesTray .tabOpenClose').removeClass('bouncing');
    setTimeout(function() {
      $('#notesTray .tabOpenClose').addClass('bouncing');
    }, 100);
  };

  var _buildCurrentStateOfNotes = function() {
    var notes = [];

    var $notes = $('#notesTray .listOfNotes .snippet');

    $notes.each(function(index, el) {
      note = {
        noteId : $(el).attr('data-note-id'),
        snippet : $(el).find('.snippetContent').text(),
        note : $(el).find('.snippetNotes').text(),
      };

      notes.push(note);
    });

    return notes;

  };

  return {
    addSnippetToNotesTray : _addSnippetToNotesTray,
  };

}();


$(document).ready(function() {

  $('.section.right').on('click', '.title, .visibleStuff', function(event) {
    $(this).parent().toggleClass('open');
  });

  $('.section.right').on('click', '.bookmark', function(event) {
    ClusterNotes.addSnippetToNotesTray($(this).parents('.snippet'));
    /* Act on the event */
  });

  $('body').on('click', '.tabOpenClose', function(event) {
    $('body').toggleClass('nav-open');
    /* Act on the event */
  });

  $('#navigation').on('click', '.firstLevel', function(event) {
    $(this).parents('.packageMenu').toggleClass('open');
    /* Act on the event */
  });

  ClusterNavigation.fetchNavigation();

  if(!urlParams.query || urlParams.query.length===0) {
    $('body').addClass('nav-open');
  } else {
    ClusterListRepresentation.buildAccordionWithData('.section.right');
  }




  $('.section.right').on('click', '.snippetLabel', function(event) {
    $(this).parents('.snippet').toggleClass('expanded');
  });

  CodeInterest.initialize({ selector: '.snippet', innerCodeContainer: '.snippetSt'  });

});
