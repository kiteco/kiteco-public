.PHONY: all clean

all: results.md

results.md: retrieved.json relevant.json
	mypy --strict results.py
	python results.py \
		--retrieved=$(PWD)/retrieved.json \
		--relevant=$(PWD)/relevant.json \
		--quip_titles=$(PWD)/docs/quip/titles.json \
		--issue_titles=$(PWD)/docs/issues/titles.json \
		--results=$(PWD)/results.md

retrieved.json: docs/quip docs/issues relevant.json
	go run ./cmds/retrieve \
		--retrieved=$(PWD)/retrieved.json \
		--root=$(PWD)/docs \
		--quip=$(PWD)/docs/quip \
		--issues=$(PWD)/docs/issues

relevant.json: docs/issues
	mypy --strict relevant.py
	python relevant.py \
		--links=$(PWD)/docs/issues/links.json \
		--relevant_issues=$(PWD)/relevant.json

docs/quip: docs relevant.json
	mypy --strict quip.py
	rm -rf docs/quip
	mkdir docs/quip
	python quip.py \
		--relevant=$(PWD)/relevant.json \
		--docs=$(PWD)/docs/quip \
		--titles=$(PWD)/docs/quip/titles.json

docs/issues: docs
	mypy --strict issues.py
	rm -rf docs/issues
	mkdir docs/issues
	python issues.py \
		--issues=$(PWD)/docs/issues \
		--titles=$(PWD)/docs/issues/titles.json \
		--links=$(PWD)/docs/issues/links.json

docs:
	mkdir docs

clean:
	rm -rf docs
	-rm retrieved.json
	-rm relevant.json
