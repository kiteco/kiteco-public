.PHONY: validate train clean

validate: model/validate.txt
	cat model/validate.txt

train: bindata.go

model/validate.txt: data/validate.csv
	python model/validate.py

data/validate.csv:
	go run ./data/cmds/validate/

bindata.go: model/normalizers.json
	mkdir serve
	cp model/normalizers.json serve/normalizers.json
	go-bindata -o bindata.go -pkg mixing serve/
	rm -rf serve

model/normalizers.json: data/train.csv
	python model/train.py

data/train.csv:
	go run ./data/cmds/train/

clean:
	-rm model/validate.txt
	-rm data/validate.csv
	-rm model/normalizers.json
	-rm data/train.csv
