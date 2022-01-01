package main

import (
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/atom"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/neovim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/spyder"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/sublime"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vscode"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func createPlugins(betaChannel bool) ([]editor.Plugin, error) {
	processMgr := process.NewManager()

	sublimeMgr, err := sublime.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	jetbrainsManagers, err := jetbrains.NewJetBrainsManagers(processMgr, betaChannel)
	if err != nil {
		return nil, err
	}

	atomMgr, err := atom.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	vscodeMgr, err := vscode.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	vimMgr, err := vim.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	neovimMgr, err := neovim.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	spyderMgr, err := spyder.NewManager(processMgr)
	if err != nil {
		return nil, err
	}

	return append([]editor.Plugin{
		sublimeMgr,
		atomMgr,
		vscodeMgr,
		vimMgr,
		neovimMgr,
		spyderMgr,
	}, jetbrainsManagers...), nil
}
