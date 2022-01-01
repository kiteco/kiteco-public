require('shelljs/global');
fs = require('fs');
path = require('path');
marked = require('marked');
mustache = require('mustache');

marked.setOptions({
  highlight: function(code) {
    return require('highlight.js').highlightAuto(code, ['python']).value;
  },
});

TEMPLATE = fs.readFileSync('docs/doc-template.html', { encoding: 'utf8' });

ls('docs/*.md').forEach(function(mdfile) {
  var outfile = 'static/docs/' + path.parse(mdfile).name + '.html';
  console.log('building', outfile);
  var markdownOutput = marked(fs.readFileSync(mdfile, { encoding: 'utf8' }));
  var match = /<h1[^>]*>(.*)<\/h1>/.exec(markdownOutput);
  var title = (match ? match[1] + ' | ' : '') + 'Kite Curation';

  var htmlOutput = mustache.render(TEMPLATE, {
    title: title,
    body: markdownOutput,
  });
  fs.writeFileSync(outfile, htmlOutput);
});
