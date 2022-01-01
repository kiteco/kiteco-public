.PHONY: all clean

all: control_examples.json treatment_examples.json control_validation.csv treatment_validation.csv
	python template.py

control_examples.json:
	bash examples.sh control true

treatment_examples.json:
	bash examples.sh treatment false

control_validation.csv:
	bash validation.sh control true

treatment_validation.csv:
	bash validation.sh treatment false

clean:
	-rm control_examples.json
	-rm control_validation.csv
	-rm treatment_examples.json
	-rm treatment_validation.csv
