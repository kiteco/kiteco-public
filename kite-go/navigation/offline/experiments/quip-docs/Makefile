.PHONY: all clean

all: results.md

results.md: retrieved.json relevant.json
	mypy --strict report.py
	python report.py \
		--retrieved=$(PWD)/retrieved.json \
		--relevant=$(PWD)/relevant.json \
		--titles=$(PWD)/docs/titles.json \
		--results=$(PWD)/results.md

retrieved.json: docs relevant.json
	go run ./cmds/retrieve \
		--relevant=$(PWD)/relevant.json \
		--retrieved=$(PWD)/retrieved.json \
		--docs=$(PWD)/docs

relevant.json: links.json pulls.json extensions
	mypy --strict relevant.py
	python relevant.py \
		--links=$(PWD)/links.json \
		--pulls=$(PWD)/pulls.json \
		--extensions=$(PWD)/extensions \
		--relevant=$(PWD)/relevant.json

docs: relevant.json
	mypy --strict docs.py
	rm -rf docs
	mkdir docs
	python docs.py \
		--relevant=$(PWD)/relevant.json \
		--docs=$(PWD)/docs \
		--titles=$(PWD)/docs/titles.json

links.json:
	mypy --strict links.py
	python links.py \
		--links=$(PWD)/links.json

pulls.json:
	mypy --strict pulls.py
	python pulls.py \
		--pulls=$(PWD)/pulls.json

extensions:
	go run ./cmds/extensions \
		--extensions=$(PWD)/extensions

clean:
	rm -rf docs
	-rm retrieved.json
	-rm relevant.json
	-rm links.json
	-rm pulls.json
	-rm extensions
