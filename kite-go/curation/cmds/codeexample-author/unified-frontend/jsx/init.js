React = require('react/addons');
ReactDOM = require('react-dom');

// real janky way to deal with the way the curation tool isn't quite a single page app
switch (INIT_DATA.view) {
case "author":
  CodeAuthoringTool = require('./code-authoring-tool.js');
  ReferenceTool = require('./reference-tool.js');
  require('./container-logic.js');

  var NavEvents = new Dispatcher();

  ReactDOM.render(
    <CodeAuthoringTool lang={INIT_DATA.language} pkg={INIT_DATA.package} userEmail={INIT_DATA.userEmail} />,
    document.getElementById('codeAuthoringTool'));

  ReactDOM.render(
    <ReferenceTool lang={INIT_DATA.language} pkg={INIT_DATA.package} />,
    document.getElementById('referenceTool'));
  break;
case "moderate":
  ModerationTool = require('./moderation-tool.js');
  ReactDOM.render(<ModerationTool />, document.body);
  break;
default:
  console.error("invalid page view:", INIT_DATA.view);
  break;
}
