GOPATH ?= ${HOME}/go
ifeq ($(origin GOPATH), undefined)
	GOPATH := ${HOME}/go
endif
export GOPATH
KITECO := github.com/kiteco/kiteco/
KITEPATH := ${GOPATH}/src/${KITECO}

analyze: asttrace.json
	${KITEPATH}/kite-exp/telemetry-analysis/asttrace/analyze.py $<

asttrace.bin:
	go build -o $@ ${KITECO}/kite-exp/telemetry-analysis/asttrace

asttrace.json: | asttrace.bin
	./asttrace.bin >$@
