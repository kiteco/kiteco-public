const WindowMode = Object.freeze({
  NORMAL: 'normal',
  FOCUS_ON_DOCS: 'focus-on-docs',
  ALWAYS_ON_TOP: 'always-on-top',
});

const isWindowModeValid = mode => {
  return Object.values(WindowMode).includes(mode);
};

const validateWindowMode = mode => {
  return isWindowModeValid(mode) ? mode : WindowMode.NORMAL;
};

module.exports = {
  WindowMode,
  isWindowModeValid,
  validateWindowMode,
};
