.PHONY = package deploy clean

ZIPFILE_NAME = telemetry-loader.zip
BUILD_DIR = $(CURDIR)/dist/
ZIPFILE_PATH = $(addprefix $(BUILD_DIR), $(ZIPFILE_NAME))

.venv: Pipfile
	PIPENV_VENV_IN_PROJECT=1 pipenv install --deploy
	touch .venv

dev: Pipfile
	pipenv install --dev

clean:
	pipenv --rm

$(BUILD_DIR):
	mkdir -p $@

package: $(ZIPFILE_PATH)
$(ZIPFILE_PATH): .venv $(shell find telemetry_loader/ -type f -name '*.py') | $(BUILD_DIR)
	cd .venv/lib/python3.7/site-packages/ && zip -r $(ZIPFILE_PATH) * 
	zip -rg $(ZIPFILE_PATH) telemetry_loader

lint:
	pipenv run flake8 --exclude=.venv

test:
	pipenv run pytest
