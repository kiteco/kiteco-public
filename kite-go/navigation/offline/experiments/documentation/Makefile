.PHONY: all clean

all: results.md

results.md: retrieved.json relevant.json
	mypy --strict report.py
	python report.py \
		--retrieved=$(PWD)/retrieved.json \
		--relevant=$(PWD)/relevant.json \
		--results=$(PWD)/results.md

retrieved.json: docs relevant.json
	go run ./cmds/retrieve \
		--relevant=$(PWD)/relevant.json \
		--retrieved=$(PWD)/retrieved.json \
		--index=$(PWD)/docs/index.json

relevant.json: git.csv extensions
	mypy --strict relevant.py
	python relevant.py \
		--git=$(PWD)/git.csv \
		--extensions=$(PWD)/extensions \
		--relevant=$(PWD)/relevant.json

docs: relevant.json
	rm -rf docs
	mkdir docs
	go run ./cmds/docs \
		--relevant=$(PWD)/relevant.json \
		--docs=$(PWD)/docs \
		--index=$(PWD)/docs/index.json

git.csv: gitlog
	bash git-show.sh gitlog > git.csv

gitlog:
	git log \
		--since=2020-01-01 \
		--until=2021-01-01 \
		--pretty=format:"%H" \
		> gitlog

extensions:
	go run ./cmds/extensions

clean:
	rm -rf docs
	-rm retrieved.json
	-rm relevant.json
	-rm git.csv
	-rm gitlog
	-rm extensions
