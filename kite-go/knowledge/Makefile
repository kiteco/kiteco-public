.PHONY: app extract clean

ifeq ($(OS), Windows_NT)
KITECO = $(GOPATH)\src\github.com\kiteco\kiteco
PULLS_DIR = $(HOME)\kite-navigation
CLOSED_PULLS_PATH = $(HOME)\.kite\github\kiteco\kiteco\pulls.json
OPEN_PULLS_DIR = $(PULLS_DIR)\open
else
KITECO = $(GOPATH)/src/github.com/kiteco/kiteco/
PULLS_DIR = $(HOME)/kite-navigation
CLOSED_PULLS_PATH = $(HOME)/.kite/github/kiteco/kiteco/pulls.json
OPEN_PULLS_DIR = $(PULLS_DIR)/open
endif

NUM_PULLS = 1000

app: server/bindata.go
	go run ./cmds/index/ --dirs $(KITECO)
	go run ./cmds/app/ --closedpullsdir=$(CLOSED_PULLS_PATH) --openpullsdir=$(OPEN_PULLS_DIR) --root=$(KITECO)

server/bindata.go: server/templates/* server/static/*
	go-bindata -pkg knowledge -o server/bindata.go server/templates/ server/static/

csindex:
	go run ./cmds/index/ --dirs $(KITECO)

extract:
	-mkdir -p $(PULLS_DIR)
	go run ./cmds/github-data/ --writedir=$(PULLS_DIR)
	@python pypulls/pulls.py $(GITHUB_AUTH_TOKEN) kiteco/kiteco

clean:
	go run ./cmds/index/
	rm -rf $(HOME)/.kite/github
	rm -rf $(PULLS_DIR)
