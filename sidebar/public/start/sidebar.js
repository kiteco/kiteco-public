const { app, BrowserWindow, ipcMain, Menu, shell } = require('electron')
const path = require('path')
const url = require('url')
const { isDev, isLocalhostDev } = require('../utils/electron-is-dev')
const { handleRendererException, handleRendererRejection, handleErrorBoundaryCatch } = require('../utils/error-handling')
const { kiteProcess } = require('../utils/kite-process')
const { metrics } = require('../utils/metrics')
const { updateProxySettings } = require('../utils/proxy')
const { WindowMode, validateWindowMode } = require('../utils/settings')
const { Domains } = require('../utils/domains')


const width = 400
const maxWidth = 480
const height = 760
const minHeight = 700

let sidebarPromise = null
function openSidebar() {
  if (!sidebarPromise) {
    sidebarPromise = new Promise(function (resolve, reject) {
      // create window & show dock icon

      let sidebar = new BrowserWindow({
        width,
        height,
        maxWidth,
        minHeight,
        minWidth: width,
        maximizable: false,
        acceptFirstMouse: true,
        autoHideMenuBar: true,
        backgroundColor: '#3818b1',
        titleBarStyle: 'hiddenInset',
        webPreferences: {
          nodeIntegration: true,
        },
      })
      if (app.dock) {
        app.dock.show()
      }
      sidebar.on('closed', () => {
        if (app.dock) {
          app.dock.hide()
        }
        sidebarPromise = null
      })


      // - set up IPC <-> React app

      ipcMain.removeAllListeners('error-boundary')
      ipcMain.on('error-boundary', handleErrorBoundaryCatch)
      ipcMain.removeAllListeners('renderer-exception')
      ipcMain.on('renderer-exception', handleRendererException)

      // TODO(naman) unused?
      ipcMain.removeAllListeners('renderer-rejection')
      ipcMain.on('renderer-rejection', handleRendererRejection)

      ipcMain.removeAllListeners('restart-kite')
      ipcMain.on('restart-kite', async (event) => {
        try {
          await kiteProcess.launchKite()
          await event.sender.send('restart-kite-success')
        } catch (err) {
          if (err === 'MOCK REJECTION') {
            event.sender.send('no-restart-support')
          } else {
            event.sender.send('restart-kite-error')
          }
        }
      })

      let windowMode = WindowMode.NORMAL
      ipcMain.removeAllListeners('set-window-mode')
      ipcMain.on('set-window-mode', (event, val) => {
        windowMode = validateWindowMode(val)
        sidebar.setAlwaysOnTop(windowMode === WindowMode.ALWAYS_ON_TOP)
      })

      ipcMain.removeAllListeners('focus-window')
      ipcMain.on('focus-window', _ => {
        if (sidebar.isMinimized) {
          sidebar.restore()
        }
        app.focus({ steal: true })
      })

      ipcMain.removeAllListeners('docs-rendered')
      ipcMain.on('docs-rendered', (event) => {
        if (windowMode === WindowMode.FOCUS_ON_DOCS && sidebar) {
          // BrowserWindow.moveTop doesn't seem to work reliably on Windows, so we
          // use setAlwaysOnTop as a workaround.
          if (process.platform === 'win32') {
            sidebar.setAlwaysOnTop(true)
            sidebar.setAlwaysOnTop(false)
          } else {
            sidebar.moveTop()
          }
        }
      })

      // TODO(naman) unused?
      ipcMain.removeAllListeners('adjust-window-size')
      ipcMain.on('adjust-window-size', (event, dim) => {
        if (sidebar) {
          const currentSizes = sidebar.getSize()
          sidebar.setSize(
            dim.width || currentSizes[0],
            dim.height || currentSizes[1],
            true
          )
        }
      })

      let proxyMode = "environment"
      let proxyURL = ""
      ipcMain.removeAllListeners('set-proxy-mode')
      ipcMain.removeAllListeners('set-proxy-url')
      ipcMain.on('set-proxy-mode', (event, mode) => {
        console.log("ipc: setting proxy mode to " + mode)
        proxyMode = mode
        if (sidebar) {
          updateProxySettings(sidebar.webContents.session, proxyMode, proxyURL)
        }
      })
      ipcMain.on('set-proxy-url', (event, url) => {
        console.log("ipc: setting proxy url to " + url)
        proxyURL = url
        if (sidebar) {
          updateProxySettings(sidebar.webContents.session, proxyMode, proxyURL)
        }
      })


      // - load React app

      sidebar.kite_os = process.platform // linux, win32, darwin

      let sidebarURL = url.format({
        pathname: path.join(__dirname, '../../build/index.html'),
        protocol: 'file:',
        slashes: true,
      })
      if (isLocalhostDev) {
        sidebarURL = 'http://localhost:3000/'
      }
      sidebar.loadURL(sidebarURL)


      // - configure menu

      sidebar.webContents.on('new-window', function (e, url) {
        // open links that would open in a new window in the browser
        e.preventDefault()
        shell.openExternal(url)
      })

      const template = [
        {
          label: 'Edit',
          submenu: [
            { role: 'undo' },
            { role: 'redo' },
            { type: 'separator' },
            { role: 'cut' },
            { role: 'copy' },
            { role: 'paste' },
            { role: 'pasteandmatchstyle' },
            { role: 'delete' },
            { role: 'selectall' },
          ],
        },
        {
          label: 'View',
          submenu: [
            { role: 'resetzoom' },
            { role: 'zoomin', accelerator: 'CommandOrControl+=' },
            { role: 'zoomout' },
            { type: 'separator' },
            { role: 'togglefullscreen' },
          ],
        },
        {
          role: 'window',
          submenu: [
            { role: 'minimize' },
            { role: 'close' },
          ],
        },
        {
          role: 'help',
          submenu: [
            {
              label: 'Learn More',
              click() {
                shell.openExternal(`http://${Domains.Help}/`)
              },
            },
          ],
        },
      ]

      if (isDev) {
        template[1].submenu.push({ role: 'reload' }, { role: 'forcereload' }, { role: 'toggledevtools' })
      }

      if (process.platform === 'darwin') {
        template.unshift({
          label: app.name,
          submenu: [
            { role: 'about' },
            { type: 'separator' },
            { role: 'services', submenu: []},
            { type: 'separator' },
            { role: 'hide' },
            { role: 'hideothers' },
            { role: 'unhide' },
            { type: 'separator' },
            { role: 'quit' },
          ],
        })

        // Edit menu
        template[1].submenu.push(
          { type: 'separator' },
          {
            label: 'Speech',
            submenu: [
              { role: 'startspeaking' },
              { role: 'stopspeaking' },
            ],
          }
        )

        // Window menu
        template[3].submenu = [
          { role: 'close' },
          { role: 'minimize' },
          { role: 'zoom' },
          { type: 'separator' },
          { role: 'front' },
        ]
      }

      menu = Menu.buildFromTemplate(template)
      Menu.setApplicationMenu(menu)
      sidebar.on('focus', () => {
        metrics.trackSidebarFocused()
      })

      sidebar.setMenuBarVisibility(false)

      // show dev tools in production build if this env variable is set
      if (process.env.ELECTRON_ENV === 'development') {
        sidebar.webContents.openDevTools()
      }


      // - resolve to sidebar
      sidebar.webContents.on('did-finish-load', () => resolve(sidebar))
    })
  }
  return sidebarPromise
}

function openURL(url) {
  openSidebar().then((sidebar) => {
    sidebar.webContents.send('transitionTo', url)
    if (sidebar.isMinimized()) {
      sidebar.restore()
    }
    sidebar.focus()
  })
}

module.exports = {
  openSidebar,
  openURL,
}
