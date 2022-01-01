const { override, fixBabelImports, addLessLoader } = require('customize-cra');

module.exports = override(
  // auto import sytles of imported ant design component (prevents an extra import statement)
  fixBabelImports('import', {
    libraryName: 'antd',
    libraryDirectory: 'es',
    style: true
  }),
  addLessLoader({
    lessOptions: {
      javascriptEnabled: true,
      modifyVars: {
        "@primary-color": "#14B4C3",
        "@link-color": "#14B4C3",
      },
    },
  })
);
