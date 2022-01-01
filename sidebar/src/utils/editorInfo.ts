import logoAtom from '../assets/editorIcons/atom.png@2x.png'
import logoEmacs from '../assets/editorIcons/emacs.png@2x.png'
import logoIntellij from '../assets/editorIcons/intellij.png@2x.png'
import logoNeovim from '../assets/editorIcons/neovim.png@2x.png'
import logoPycharm from '../assets/editorIcons/pycharm.png@2x.png'
import logoGoland from '../assets/editorIcons/goland.png@2x.png'
import logoWebstorm from '../assets/editorIcons/webstorm.png@2x.png'
import logoPhpstorm from '../assets/editorIcons/phpstorm@2x.png'
import logoClion from '../assets/editorIcons/clion@2x.png'
import logoRubymine from '../assets/editorIcons/rubymine@2x.png'
import logoRider from '../assets/editorIcons/rider@2x.png'
import logoAppcode from '../assets/editorIcons/appcode@2x.png'
import logoAndroidStudio from '../assets/editorIcons/android-studio@2x.png'
import logoSublime from '../assets/editorIcons/sublime.png@2x.png'
import logoVim from '../assets/editorIcons/vimlogo.png@2x.png'
import logoVscode from '../assets/editorIcons/vscode.png@2x.png'
import logoSpyder from '../assets/editorIcons/spyder.png@2x.png'
import logoJupyterlab from '../assets/editorIcons/jupyterlab.png@2x.png'

export const getIconForEditor = (name: string): any => {
  const iconMap: { [name: string]: any } = {
    "atom": logoAtom,
    "emacs": logoEmacs,
    "intellij": logoIntellij,
    "neovim": logoNeovim,
    "pycharm": logoPycharm,
    "goland": logoGoland,
    "webstorm": logoWebstorm,
    "clion": logoClion,
    "phpstorm": logoPhpstorm,
    "rubymine": logoRubymine,
    "rider": logoRider,
    "appcode": logoAppcode,
    "android-studio": logoAndroidStudio,
    "sublime": logoSublime,
    "sublime3": logoSublime,
    "vim": logoVim,
    "vscode": logoVscode,
    "spyder": logoSpyder,
    "jupyterlab": logoJupyterlab,
  }
  return iconMap[name]
}

export const getReadableNameForEditor = (name: string): string => {
  const readableNameMap: { [name: string]: string } = {
    "atom": "Atom",
    "emacs": "Emacs",
    "intellij": "Intellij",
    "neovim": "Neovim",
    "pycharm": "PyCharm",
    "goland": "GoLand",
    "webstorm": "WebStorm",
    "clion": "CLion",
    "phpstorm": "PhpStorm",
    "rubymine": "RubyMine",
    "rider": "Rider",
    "appcode": "AppCode",
    "android-studio": "Android Studio",
    "sublime": "Sublime",
    "sublime3": "Sublime",
    "vim": "Vim",
    "vscode": "VSCode",
    "spyder": "Spyder",
    "jupyterlab": "JupyterLab",
  }
  return readableNameMap[name]
}
