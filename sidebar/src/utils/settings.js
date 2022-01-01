const { remote } = window.require('electron');
const settings = remote.require('./utils/settings');

module.exports = {
  WindowMode: settings.WindowMode,
  isWindowModeValid: settings.isWindowModeValid,
  validateWindowMode: settings.validateWindowMode,
};
