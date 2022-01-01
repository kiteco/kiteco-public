const { BrowserWindow, screen, shell } = require('electron')
const path = require('path');
const url = require('url');

let notifWindow = null;
function openNotification(name) {
  if (!name) {
    return;
  }
  if (notifWindow) {
    return;
  }

  let width = 480;
  let height = 150;
  let margin = 20;

  let x, y;
  let type;
  let display = screen.getPrimaryDisplay();
  switch (process.platform) {
    case 'darwin':
      // top right
      x = display.bounds.width - width - margin;
      y = margin + 21; // account for menu bar
      break
    case 'win32':
      // bottom right
      x = display.bounds.width - width - margin;
      y = display.bounds.height - height - margin - 32; // account for taskbar
      break
    case 'linux':
      type = 'notification'
      // top right
      x = display.bounds.width - width - margin;
      y = margin;
      break
  }

  notifWindow = new BrowserWindow({
    x,
    y,
    width,
    height,
    frame: false,
    resizable: false,
    maximizable: false,
    focusable: false,
    alwaysOnTop: true,
    fullscreenable: false,
    skipTaskbar: true,
    show: false,
    acceptFirstMouse: true,
    autoHideMenuBar: true,
    transparent: true,
    type,
    webPreferences: {
      nodeIntegration: true,
    }
  });
  notifWindow.setVisibleOnAllWorkspaces(true);
  notifWindow.on('closed', () => notifWindow = null);

  // this is linux, win32, darwin
  notifWindow.kite_os = process.platform;

  let notifUrl = url.format({
    hostname: 'localhost',
    port: 46624,
    pathname: path.posix.join('clientapi/notifications', name),
    protocol: 'http:',
    slashes: true,
  });
  notifWindow.loadURL(notifUrl);

  // open links that would open in a new window
  // in system default browser
  notifWindow.webContents.on('new-window', function (e, url) {
    e.preventDefault();
    shell.openExternal(url);
  });

  notifWindow.showInactive()
}

module.exports = {
  openNotification,
}
