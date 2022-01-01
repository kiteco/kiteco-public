.PHONY: clean

README.md: params.json report.py report.txt
	python report.py

params.json: train.csv model.py
	-mkdir charts
	python model.py

train.csv: data.py query.sql
	python data.py

clean:
	rm -rf charts
	-rm train.csv
