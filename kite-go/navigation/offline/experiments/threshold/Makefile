.PHONY: all clean

MAX_FILE_RECS = 100

all: commits.png text.png

commits.png: commits.csv charts.py
	mypy --strict charts.py
	python charts.py \
		--records=commits.csv \
		--hits=commits-hits.png \
		--cdf=commits-cdf.png

text.png: text.csv charts.py
	mypy --strict charts.py
	python charts.py \
		--records=text.csv \
		--hits=text-hits.png \
		--cdf=text-cdf.png

commits.csv:
	go run ../../cmds/validate \
		--repospath=../../validation/repos \
		--readdir=$(HOME)/nav-validation \
		--recordspath=commits.csv \
		--maxfilerecs=$(MAX_FILE_RECS) \
		--skiplines \
		--usecommits

text.csv:
	go run ../../cmds/validate \
		--repospath=../../validation/repos \
		--readdir=$(HOME)/nav-validation \
		--recordspath=text.csv \
		--maxfilerecs=$(MAX_FILE_RECS) \
		--skiplines

clean:
	-rm commits.csv
	-rm text.csv
