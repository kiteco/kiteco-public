# Kite LSP

`kite-lsp` is an intermediary between editor-clients that speak [Language Server Protocol](https://microsoft.github.io/language-server-protocol/), and the Kite Engine. It maintains an LSP session, and translates between LSP requests and Kite API requests.

## Setup with JupyterLab
First, make sure you've installed [JupyterLab](https://github.com/jupyterlab/jupyterlab), [`jupyterlab-kite`](https://github.com/kiteco/jupyterlab-kite#installation), and the Kite Engine.

Then, to build `kite-lsp` run the following:
```bash
go install github.com/kiteco/kiteco/kite-go/lsp/cmds/kite-lsp
```
This will build the `kite-lsp` binary and then place it in `$GOPATH/bin`.

To make `jupyterlab-kite` aware of your build of `kite-lsp`, move to your Jupyter config folder (usually `$HOME/.jupyter`) and create a file called `jupyter_notebook_config.json`, with the following contents:
```json
{
  "LanguageServerManager": {
      "language_servers": {
          "kitels": {
              "argv": [
                  "YOUR_KITE_LSP_LOCATION"
              ],
              "languages": [
                  "python"
              ],
              "version": 2
          }
      }
  }
}
```

If you built `kite-lsp` using the `go install` instruction above, then `YOUR_KITE_LSP_LOCATION` will be your `GOPATH` plus `bin/kite-lsp`, and this needs to be provided as an absolute path (i.e. don't just put `"$GOPATH/bin/kite-lsp"` in the config).

## Usage
After installation, make sure that you have the Kite Engine running on your machine, and then start JupyterLab by running:
```bash
jupyter lab
```

