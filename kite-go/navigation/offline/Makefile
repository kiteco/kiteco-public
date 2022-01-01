.PHONY: clean github-read github-delete

VALIDATION_DIR = $(HOME)/nav-validation

README.md: validation.csv performance.csv
	python template.py

validation.csv:
	go run ./cmds/validate/ \
		--repospath=validation/repos \
		--readdir=$(HOME)/nav-validation \
		--statspath=validation.csv \
		--usecommits
	column -s=, -t validation.csv

performance.csv:
	go run ./cmds/performance --resultspath=$(PWD)/performance.csv
	column -s=, -t performance.csv

clean:
	-rm validation.csv
	-rm performance.csv

github-read:
	bash validation/clone.sh $(VALIDATION_DIR) validation/repos
	go run ./cmds/github-data/ --repospath=$(PWD)/validation/repos --writedir=$(VALIDATION_DIR)

github-delete:
	rm -rf $(VALIDATION_DIR)
